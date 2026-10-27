package packs

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/skills"
)

// InjectLine describes one injected node for TUI / logs.
type InjectLine struct {
	NodeID  string
	Title   string
	Reason  string
}

// AssembleResult is the deterministic pack injection for one turn.
type AssembleResult struct {
	SystemAppendix string
	UserPreface    string
	Lines          []InjectLine
}

// AssembleRequest configures pack assembly.
type AssembleRequest struct {
	Goal           string
	RecentText     string
	PacksAllow     []string
	SkillLoader    *skills.Loader
	ResolveSkills  []string
}

var (
	reContextChip = regexp.MustCompile(`(?i)\[Context:\s*([^\]]+)\]`)
	reSkillChip   = regexp.MustCompile(`(?i)\[Skill:\s*([^\]]+)\]`)
	reAtMention   = regexp.MustCompile(`@([a-zA-Z0-9][a-zA-Z0-9_./:-]*)`)
)

// Assemble runs Always → explicit mentions → keyword (exact, then BM25) → semantic RAG → manual catalog.
func (idx *Index) Assemble(ctx context.Context, req AssembleRequest) (AssembleResult, error) {
	out := AssembleResult{}
	if idx == nil {
		return out, nil
	}
	nodes := idx.filterNodes(req.PacksAllow)
	nodes = withoutLegacyAgentsMD(nodes)
	if len(nodes) == 0 {
		return out, nil
	}

	budget := idx.cfg.MaxCharsOrDefault()
	used := 0
	injected := map[string]bool{}
	var systemBlocks []string
	var userBlocks []string

	appendNode := func(n PackNode, reason string, targetUser bool) {
		id := n.NodeID()
		if injected[id] {
			return
		}
		block := formatNodeBlock(n)
		if block == "" {
			return
		}
		if used+len(block) > budget && budget > 0 && n.Legacy != LegacyAgentsMD {
			return
		}
		injected[id] = true
		used += len(block)
		out.Lines = append(out.Lines, InjectLine{NodeID: id, Title: n.DisplayName(), Reason: reason})
		if targetUser {
			userBlocks = append(userBlocks, block)
		} else {
			systemBlocks = append(systemBlocks, block)
		}
	}

	// 1. Always
	always := filterMode(nodes, ModeAlways)
	sortByPriority(always)
	for _, n := range always {
		appendNode(n, "always", false)
	}

	// 2. Explicit mentions / chips
	for _, n := range resolveExplicit(nodes, req) {
		appendNode(n, "explicit", true)
	}

	haystack := strings.ToLower(strings.TrimSpace(req.Goal + "\n" + req.RecentText))

	// 3. Keyword exact
	for _, n := range KeywordExactHits(filterMode(nodes, ModeKeyword), haystack) {
		appendNode(n, keywordReason(n, haystack), false)
	}

	// 3b. Keyword BM25 fallback
	if idx.cfg.KeywordBM25OrDefault() {
		remaining := filterMode(nodes, ModeKeyword)
		var candidates []PackNode
		for _, n := range remaining {
			if !injected[n.NodeID()] {
				candidates = append(candidates, n)
			}
		}
		for _, hit := range SearchBM25(candidates, req.Goal, idx.cfg.MinBM25ScoreOrDefault()) {
			appendNode(hit.Node, fmt.Sprintf("keyword:bm25 (%.2f)", hit.Score), false)
		}
	}

	// 4. Semantic RAG
	if idx.rag != nil && idx.cfg.SemanticOrDefault() {
		semanticNodes := filterMode(nodes, ModeSemantic)
		if len(semanticNodes) > 0 && used < budget {
			hits, err := idx.rag.Query(ctx, rag.QueryRequest{
				Collection: idx.Collection(),
				Query:      strings.TrimSpace(req.Goal),
				TopK:       idx.cfg.SemanticTopKOrDefault(),
				MinScore:   idx.cfg.MinSemanticScoreOrDefault(),
			})
			if err == nil {
				for _, hit := range hits {
					n, ok := nodeFromRAGHit(semanticNodes, hit.Source)
					if !ok {
						continue
					}
					chunk := strings.TrimSpace(hit.Text)
					if chunk == "" {
						continue
					}
					id := n.NodeID() + "#rag"
					if injected[id] {
						continue
					}
					block := fmt.Sprintf("## Context pack (semantic): %s\n\n%s", n.DisplayName(), chunk)
					if used+len(block) > budget && budget > 0 {
						break
					}
					injected[id] = true
					used += len(block)
					systemBlocks = append(systemBlocks, block)
					out.Lines = append(out.Lines, InjectLine{NodeID: n.NodeID(), Title: n.DisplayName(), Reason: "semantic:rag"})
				}
			}
		}
	}

	// 5. Manual catalog (names only — full load stays on invoke_skill / decide)
	manual := filterMode(nodes, ModeManual)
	if len(manual) > 0 {
		var b strings.Builder
		b.WriteString("## Context pack (manual)\n\n")
		b.WriteString("These entries load on demand via tools; names and descriptions only:\n")
		for _, n := range manual {
			line := "- " + n.DisplayName()
			if d := strings.TrimSpace(n.Meta.Description); d != "" {
				line += ": " + d
			} else if w := strings.TrimSpace(n.Meta.When); w != "" {
				line += ": " + w
			}
			b.WriteString(line + "\n")
		}
		systemBlocks = append(systemBlocks, strings.TrimSpace(b.String()))
	}

	out.SystemAppendix = strings.Join(systemBlocks, "\n\n")
	out.UserPreface = strings.Join(userBlocks, "\n\n")
	return out, nil
}

func (idx *Index) filterNodes(allow []string) []PackNode {
	var out []PackNode
	for _, n := range idx.Nodes() {
		if NodeAllowed(n, allow) {
			out = append(out, n)
		}
	}
	return out
}

// AGENTS.md is injected via agentsdoc in the agent loop (no pack char budget).
func withoutLegacyAgentsMD(nodes []PackNode) []PackNode {
	if len(nodes) == 0 {
		return nodes
	}
	out := make([]PackNode, 0, len(nodes))
	for _, n := range nodes {
		if n.Legacy == LegacyAgentsMD {
			continue
		}
		out = append(out, n)
	}
	return out
}

func filterMode(nodes []PackNode, mode InjectionMode) []PackNode {
	var out []PackNode
	for _, n := range nodes {
		if n.EffectiveInjection() == mode {
			out = append(out, n)
		}
	}
	return out
}

func sortByPriority(nodes []PackNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Meta.Priority > nodes[j].Meta.Priority
	})
}

func formatNodeBlock(n PackNode) string {
	body := strings.TrimSpace(n.Body)
	if body == "" {
		return ""
	}
	title := n.DisplayName()
	if n.PackName != "" {
		title = n.PackName + "/" + title
	}
	return fmt.Sprintf("## Context pack: %s\n\n%s", title, body)
}

func keywordReason(n PackNode, haystack string) string {
	for _, term := range keywordTerms(n) {
		t := strings.ToLower(strings.TrimSpace(term))
		if t != "" && strings.Contains(haystack, t) {
			return "keyword:" + term
		}
	}
	return "keyword"
}

func resolveExplicit(nodes []PackNode, req AssembleRequest) []PackNode {
	var out []PackNode
	seen := map[string]bool{}
	add := func(n PackNode) {
		id := n.NodeID()
		if seen[id] {
			return
		}
		seen[id] = true
		out = append(out, n)
	}

	text := req.Goal
	for _, m := range reContextChip.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		if n, ok := findNode(nodes, strings.TrimSpace(m[1])); ok {
			add(n)
		}
	}
	for _, m := range reAtMention.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		ref := strings.TrimSpace(m[1])
		if n, ok := findNode(nodes, ref); ok {
			add(n)
		}
	}
	for _, m := range reSkillChip.FindAllStringSubmatch(text, -1) {
		name := strings.TrimSpace(m[1])
		if n, ok := findNode(nodes, name); ok {
			add(n)
			continue
		}
		if req.SkillLoader != nil {
			all, err := req.SkillLoader.LoadAll()
			if err == nil {
				for _, s := range all {
					if strings.EqualFold(s.Name, name) {
						node, err := LoadLegacySkillNode(s.Path, "skills")
						if err == nil {
							add(node)
						}
						break
					}
				}
			}
		}
	}
	for _, name := range req.ResolveSkills {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if n, ok := findNode(nodes, name); ok {
			add(n)
		}
	}
	return out
}

func findNode(nodes []PackNode, ref string) (PackNode, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return PackNode{}, false
	}
	refLower := strings.ToLower(ref)
	for _, n := range nodes {
		if strings.EqualFold(n.NodeID(), ref) || strings.EqualFold(n.DisplayName(), ref) {
			return n, true
		}
		if strings.EqualFold(n.PackName+"/"+n.RelPath, ref) {
			return n, true
		}
	}
	for _, n := range nodes {
		if strings.EqualFold(strings.ToLower(n.RelPath), refLower) {
			return n, true
		}
	}
	return PackNode{}, false
}

func nodeFromRAGHit(nodes []PackNode, source string) (PackNode, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return PackNode{}, false
	}
	for _, n := range nodes {
		if n.Path == source {
			return n, true
		}
		if strings.HasSuffix(n.Path, source) {
			return n, true
		}
	}
	base := strings.TrimSuffix(filepathBase(source), filepathExt(source))
	for _, n := range nodes {
		if strings.EqualFold(n.DisplayName(), base) {
			return n, true
		}
	}
	return PackNode{}, false
}

func filepathBase(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func filepathExt(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i:]
	}
	return ""
}

// FormatInjectSummary is a one-line TUI status (⚡ Injected: …).
func FormatInjectSummary(lines []InjectLine) string {
	if len(lines) == 0 {
		return ""
	}
	parts := make([]string, 0, len(lines))
	for i, line := range lines {
		if i >= 6 {
			parts = append(parts, fmt.Sprintf("+%d more", len(lines)-i))
			break
		}
		s := line.Title
		if line.Reason != "" {
			s += " (" + line.Reason + ")"
		}
		parts = append(parts, s)
	}
	return "⚡ Injected: " + strings.Join(parts, ", ")
}
