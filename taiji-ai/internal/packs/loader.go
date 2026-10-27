package packs

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Loader scans configured pack roots.
type Loader struct {
	dirs []string
}

// NewLoader creates a pack loader.
func NewLoader(dirs []string) *Loader {
	return &Loader{dirs: dirs}
}

// LoadAll returns packs and standalone legacy nodes (AGENTS.md) from all roots.
func (l *Loader) LoadAll(workDir string) ([]Pack, []PackNode, error) {
	var packs []Pack
	var legacy []PackNode
	seenPack := map[string]bool{}

	for _, root := range l.dirs {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			packDir := filepath.Join(root, e.Name())
			key := packDir
			if seenPack[key] {
				continue
			}
			seenPack[key] = true
			nodes, err := loadPackDir(e.Name(), packDir)
			if err != nil {
				return nil, nil, err
			}
			if len(nodes) > 0 {
				packs = append(packs, Pack{Name: e.Name(), Dir: packDir, Nodes: nodes})
			}
		}
	}

	if node, ok, err := loadAgentsNode(workDir); err != nil {
		return nil, nil, err
	} else if ok {
		legacy = append(legacy, node)
	}
	return packs, legacy, nil
}

func loadPackDir(packName, dir string) ([]PackNode, error) {
	var nodes []PackNode
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		node, err := LoadNodeFile(packName, dir, path, LegacyNone)
		if err != nil {
			return err
		}
		nodes = append(nodes, node)
		return nil
	})
	return nodes, err
}

// LoadNodeFile reads one markdown file into a PackNode.
func LoadNodeFile(packName, packDir, path string, legacy LegacyKind) (PackNode, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return PackNode{}, err
	}
	meta, body, err := ParseDocument(string(raw))
	if err != nil {
		return PackNode{}, err
	}
	ApplyLegacyDefaults(&meta, legacy)
	rel, _ := filepath.Rel(packDir, path)
	rel = filepath.ToSlash(rel)
	if meta.Name == "" && legacy == LegacyNone {
		meta.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return PackNode{
		PackName: packName,
		RelPath:  rel,
		Path:     path,
		Body:     body,
		Meta:     meta,
		Legacy:   legacy,
	}, nil
}

func loadAgentsNode(workDir string) (PackNode, bool, error) {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return PackNode{}, false, nil
	}
	path := filepath.Join(workDir, "AGENTS.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PackNode{}, false, nil
		}
		return PackNode{}, false, err
	}
	body := strings.TrimSpace(string(raw))
	if body == "" {
		return PackNode{}, false, nil
	}
	meta := NodeFrontmatter{Name: "AGENTS.md", Injection: ModeAlways, Priority: 100}
	ApplyLegacyDefaults(&meta, LegacyAgentsMD)
	return PackNode{
		PackName: "",
		RelPath:  "AGENTS.md",
		Path:     path,
		Body:     body,
		Meta:     meta,
		Legacy:   LegacyAgentsMD,
	}, true, nil
}

// LoadLegacySkillNode adapts a skill directory SKILL.md as a pack node.
func LoadLegacySkillNode(skillPath, packName string) (PackNode, error) {
	dir := filepath.Dir(skillPath)
	node, err := LoadNodeFile(packName, dir, skillPath, LegacySkill)
	if err != nil {
		return PackNode{}, err
	}
	if node.Meta.Name == "" {
		node.Meta.Name = filepath.Base(dir)
	}
	return node, nil
}

// LoadLegacyDecisionNode adapts DECISION.md.
func LoadLegacyDecisionNode(path, packName string) (PackNode, error) {
	dir := filepath.Dir(path)
	node, err := LoadNodeFile(packName, dir, path, LegacyDecision)
	if err != nil {
		return PackNode{}, err
	}
	if node.Meta.Name == "" {
		node.Meta.Name = filepath.Base(dir)
	}
	return node, nil
}

// FlattenNodes returns all nodes from packs plus legacy extras.
func FlattenNodes(packs []Pack, legacy []PackNode) []PackNode {
	out := make([]PackNode, 0)
	for _, p := range packs {
		out = append(out, p.Nodes...)
	}
	out = append(out, legacy...)
	return out
}

// PackAllowed reports whether a pack name passes an allowlist (empty = all).
func PackAllowed(name string, allow []string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}
	if len(allow) == 0 {
		return true
	}
	for _, a := range allow {
		if strings.EqualFold(strings.TrimSpace(a), name) {
			return true
		}
	}
	return false
}

// NodeAllowed filters nodes by pack allowlist.
func NodeAllowed(n PackNode, allow []string) bool {
	if n.Legacy != LegacyNone {
		return true
	}
	return PackAllowed(n.PackName, allow)
}
