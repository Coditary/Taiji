package memory

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

// Service coordinates memory storage and RAG recall.
type Service struct {
	cfg           taijicfg.MemoryConfig
	store         *Store
	rag           *rag.Service
	synonymGroups [][]string
	kinds         *KindCatalog
}

// NewService builds a memory service from Taiji config.
func NewService(cfg *taijicfg.Config, ragSvc *rag.Service) *Service {
	if cfg == nil {
		return nil
	}
	memCfg := cfg.ResolvedMemory()
	return &Service{
		cfg: memCfg,
		store: NewStore(
			taijicfg.MemoryGlobalDir(cfg),
			taijicfg.ProjectMemoryDir(cfg.Root),
		),
		rag:           ragSvc,
		synonymGroups: LoadSynonymGroups(taijicfg.MemorySynonymsPath(cfg)),
		kinds:         LoadKindCatalog(cfg),
	}
}

// KindCatalog returns the merged memory kind definitions.
func (s *Service) KindCatalog() *KindCatalog {
	if s == nil {
		return nil
	}
	return s.kinds
}

// Enabled reports whether memory is active.
func (s *Service) Enabled() bool {
	return s != nil && s.cfg.EnabledOrDefault()
}

// Collection returns the RAG collection used for memory recall.
func (s *Service) Collection() string {
	if s == nil || strings.TrimSpace(s.cfg.RAGCollection) == "" {
		return "memory"
	}
	return strings.TrimSpace(s.cfg.RAGCollection)
}

// MinRecallScore returns the configured minimum RAG score.
func (s *Service) MinRecallScore() float32 {
	if s == nil || s.cfg.MinRecallScore <= 0 {
		return 0.72
	}
	return s.cfg.MinRecallScore
}

// AutoRecallOnError reports whether tool errors should trigger automatic recall hints.
func (s *Service) AutoRecallOnError() bool {
	return s != nil && s.cfg.AutoRecallOnErrorOrDefault()
}

// DescribeDirs returns a short summary of memory locations.
func (s *Service) DescribeDirs() string {
	if s == nil || s.store == nil {
		return ""
	}
	return fmt.Sprintf("local=%s global=%s collection=%s",
		s.store.projectDir, s.store.globalDir, s.Collection())
}

// Recall searches memory entries via fingerprint, BM25, and RAG.
func (s *Service) Recall(ctx context.Context, req RecallRequest) ([]Entry, error) {
	if !s.Enabled() {
		return nil, errDisabled
	}
	query := strings.TrimSpace(req.Query)
	if query == "" && strings.TrimSpace(req.Fingerprint) == "" {
		return nil, errQueryRequired
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}

	computed := Fingerprint(req.Fingerprint)
	if computed == "" {
		computed = Fingerprint(query)
	}

	all, err := s.store.ListAll()
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var out []Entry

	for _, e := range all {
		if !MatchesKind(req.Kind, e.Kind) {
			continue
		}
		if computed != "" && e.Fingerprint != "" && MatchesFingerprint(e.Fingerprint, computed) {
			copy := e
			copy.Score = 1 + MatchQuality(e.Fingerprint, computed)
			copy.MatchSource = "exact"
			key := copy.Scope + "\x00" + copy.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, copy)
		}
	}

	searchQuery := strings.TrimSpace(req.Query)
	for _, hit := range searchBM25(all, searchQuery, req.Kind, s.synonymGroups) {
		key := hit.Scope + "\x00" + hit.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, hit)
	}

	if s.rag != nil {
		ragQuery := buildRAGQuery(req, computed)
		hits, err := s.rag.Query(ctx, rag.QueryRequest{
			Collection: s.Collection(),
			Query:      ragQuery,
			TopK:       topK * 2,
			MinScore:   s.MinRecallScore(),
		})
		if err == nil {
			for _, hit := range hits {
				entry, ok := entryFromHit(all, hit.Source)
				if !ok {
					continue
				}
				if !MatchesKind(req.Kind, entry.Kind) {
					continue
				}
				key := entry.Scope + "\x00" + entry.ID
				if _, exists := seen[key]; exists {
					for i := range out {
						if out[i].Scope+"\x00"+out[i].ID != key {
							continue
						}
						if hit.Score > out[i].Score {
							out[i].Score = hit.Score
						}
						if out[i].MatchSource == "bm25" {
							out[i].MatchSource = "bm25+rag"
						}
						break
					}
					continue
				}
				seen[key] = struct{}{}
				entry.Score = hit.Score
				entry.MatchSource = "rag"
				out = append(out, entry)
			}
		}
	}

	sortEntries(out)
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

// Remember creates a new memory entry and indexes it.
func (s *Service) Remember(ctx context.Context, req RememberRequest) (Entry, error) {
	if !s.Enabled() {
		return Entry{}, errDisabled
	}
	kind, err := s.parseKind(req.Kind)
	if err != nil {
		return Entry{}, err
	}
	recallText := strings.TrimSpace(req.RecallText)
	if recallText == "" {
		return Entry{}, errRecallRequired
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = deriveMemoryTitle(kind, recallText)
	}
	verifiedFix, err := resolveVerifiedFix(s.kinds, kind, recallText, req.VerifiedFix)
	if err != nil {
		return Entry{}, err
	}
	fp := strings.TrimSpace(req.Fingerprint)
	if fp == "" && s.kinds.AutoFingerprint(kind) {
		fp = Fingerprint(recallText)
	}
	tags, keywords, recallQueries := enrichMemoryEntry(
		s.kinds, kind, title, recallText, verifiedFix,
		req.Tags, req.Keywords, req.RecallQueries,
	)
	entry := Entry{
		Kind:          kind,
		Title:         title,
		Fingerprint:   fp,
		Tags:          tags,
		Keywords:      keywords,
		RecallQueries: recallQueries,
		RecallText:    recallText,
		VerifiedFix:   verifiedFix,
		LastVerified:  todayISO(),
		RevisionLog:   appendRevisionLog("", "Created memory entry"),
	}
	saved, err := s.store.Save(req.Scope, entry)
	if err != nil {
		return Entry{}, err
	}
	if err := s.indexEntry(ctx, saved.Path); err != nil {
		saved.RevisionLog = appendRevisionLog(saved.RevisionLog, "RAG index failed: "+err.Error())
	}
	return saved, nil
}

// Revise updates an existing memory entry.
func (s *Service) Revise(ctx context.Context, req ReviseRequest) (Entry, error) {
	if !s.Enabled() {
		return Entry{}, errDisabled
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return Entry{}, errIDRequired
	}
	note := strings.TrimSpace(req.RevisionNote)
	if note == "" {
		return Entry{}, errNoteRequired
	}
	entry, err := s.store.FindByID(id)
	if err != nil {
		return Entry{}, err
	}
	if text := strings.TrimSpace(req.RecallText); text != "" {
		entry.RecallText = text
	}
	if fix := strings.TrimSpace(req.VerifiedFix); fix != "" {
		entry.VerifiedFix = fix
	}
	if len(req.Keywords) > 0 {
		entry.Keywords = dedupeTerms(append(entry.Keywords, req.Keywords...))
	}
	if len(req.RecallQueries) > 0 {
		entry.RecallQueries = dedupePhrases(append(entry.RecallQueries, req.RecallQueries...))
	}
	tags, keywords, recallQueries := enrichMemoryEntry(
		s.kinds, entry.Kind, entry.Title, entry.RecallText, entry.VerifiedFix,
		entry.Tags, entry.Keywords, entry.RecallQueries,
	)
	entry.Tags = tags
	entry.Keywords = keywords
	entry.RecallQueries = recallQueries
	switch strings.ToLower(strings.TrimSpace(req.Outcome)) {
	case "success", "hit":
		entry.Hits++
		entry.LastVerified = todayISO()
	case "miss", "failure", "failed":
		entry.Misses++
	default:
		entry.LastVerified = todayISO()
	}
	entry.RevisionLog = appendRevisionLog(entry.RevisionLog, note)
	saved, err := s.store.Save(entry.Scope, entry)
	if err != nil {
		return Entry{}, err
	}
	if err := s.indexEntry(ctx, saved.Path); err != nil {
		return saved, fmt.Errorf("saved entry but rag index failed: %w", err)
	}
	return saved, nil
}

// FormatRecallResults renders recall hits for tool output or agent hints.
func FormatRecallResults(entries []Entry) string {
	if len(entries) == 0 {
		return "no memory entries matched"
	}
	parts := make([]string, 0, len(entries))
	for i, e := range entries {
		parts = append(parts, fmt.Sprintf("--- match %d ---\n%s", i+1, e.formatRecallBlock()))
	}
	return strings.Join(parts, "\n\n")
}

// FormatErrorHint recalls memory for a tool error and returns an agent hint block.
func (s *Service) FormatErrorHint(ctx context.Context, toolName, errorText string) string {
	hint, _ := s.FormatErrorHintWithID(ctx, toolName, errorText)
	return hint
}

// FormatErrorHintWithID returns the hint and the top matched entry id.
func (s *Service) FormatErrorHintWithID(ctx context.Context, toolName, errorText string) (string, string) {
	if s == nil || !s.Enabled() || !s.AutoRecallOnError() {
		return "", ""
	}
	errorText = strings.TrimSpace(errorText)
	if errorText == "" {
		return "", ""
	}
	errorText = strings.TrimPrefix(errorText, "error:")
	errorText = strings.TrimSpace(errorText)
	entries := s.recallForToolError(ctx, toolName, errorText)
	if len(entries) == 0 {
		return "", ""
	}
	return strings.TrimSpace("[memory] Relevant entries found. Verify the match before applying.\n\n" + FormatRecallResults(entries)), entries[0].ID
}

func (s *Service) recallForToolError(ctx context.Context, toolName, errorText string) []Entry {
	kinds := []string{string(KindError), string(KindPattern), string(KindAntiPattern)}
	if s.kinds != nil {
		kinds = s.kinds.ErrorRecallKindNames()
	}
	seen := map[string]struct{}{}
	var merged []Entry
	for _, kind := range kinds {
		entries, err := s.Recall(ctx, RecallRequest{
			Query:       errorText,
			Kind:        kind,
			Fingerprint: errorText,
			ToolName:    toolName,
			TopK:        5,
		})
		if err != nil {
			continue
		}
		for _, e := range entries {
			key := e.Scope + "\x00" + e.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, e)
		}
	}
	sortEntries(merged)
	if len(merged) > 3 {
		merged = merged[:3]
	}
	return merged
}

func (s *Service) parseKind(raw string) (Kind, error) {
	if s != nil && s.kinds != nil {
		return s.kinds.ParseKind(raw)
	}
	return ParseKind(raw)
}

func (s *Service) indexEntry(ctx context.Context, path string) error {
	if s.rag == nil || strings.TrimSpace(path) == "" {
		return nil
	}
	source := ragIndexPathForEntry(path)
	if source == "" {
		source = path
	}
	_, err := s.rag.Index(ctx, rag.IndexRequest{
		Collection:  s.Collection(),
		SourcePaths: []string{source},
	})
	return err
}

func buildRAGQuery(req RecallRequest, fingerprint string) string {
	parts := []string{}
	if kind := strings.TrimSpace(req.Kind); kind != "" {
		parts = append(parts, "kind:"+kind)
	}
	if tool := strings.TrimSpace(req.ToolName); tool != "" {
		parts = append(parts, "tool:"+tool)
	}
	if fingerprint != "" {
		parts = append(parts, fingerprint)
	}
	if q := strings.TrimSpace(req.Query); q != "" {
		parts = append(parts, q)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func entryFromHit(all []Entry, source string) (Entry, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return Entry{}, false
	}
	for _, e := range all {
		if e.Path == source {
			return e, true
		}
		if ragIndexPathForEntry(e.Path) == source {
			return e, true
		}
	}
	base := filepath.Base(source)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	for _, e := range all {
		if e.ID == base {
			return e, true
		}
		if strings.HasSuffix(e.Path, base) {
			return e, true
		}
	}
	return Entry{}, false
}

func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score == entries[j].Score {
			if entries[i].Scope == entries[j].Scope {
				return entries[i].ID < entries[j].ID
			}
			return entries[i].Scope == "local"
		}
		return entries[i].Score > entries[j].Score
	})
}
