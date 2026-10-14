package skills

import (
	"context"
	"path/filepath"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

// Catalog indexes and searches invokable skills via BM25 and optional RAG.
type Catalog struct {
	loader *Loader
	rag    *rag.Service
	cfg    taijicfg.SkillsRAGConfig
}

// NewCatalog builds a skill search catalog.
func NewCatalog(loader *Loader, ragSvc *rag.Service, cfg *taijicfg.Config) *Catalog {
	if loader == nil {
		return nil
	}
	var ragCfg taijicfg.SkillsRAGConfig
	if cfg != nil {
		ragCfg = cfg.ResolvedSkillsRAG()
	} else {
		ragCfg = taijicfg.DefaultSkillsRAGConfig()
	}
	return &Catalog{loader: loader, rag: ragSvc, cfg: ragCfg}
}

// Enabled reports whether skill search is active.
func (c *Catalog) Enabled() bool {
	return c != nil && c.cfg.EnabledOrDefault()
}

// PrefetchEnabled reports whether turn-start skill search should run.
func (c *Catalog) PrefetchEnabled() bool {
	return c.Enabled() && c.cfg.PrefetchOrDefault()
}

// Collection returns the RAG collection name for skills.
func (c *Catalog) Collection() string {
	if c == nil || strings.TrimSpace(c.cfg.Collection) == "" {
		return "skills"
	}
	return strings.TrimSpace(c.cfg.Collection)
}

// EnsureIndexed indexes all invokable skills into the RAG collection.
func (c *Catalog) EnsureIndexed(ctx context.Context) error {
	if c == nil || !c.Enabled() || c.rag == nil {
		return nil
	}
	all, err := c.loader.LoadAll()
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(all))
	for _, s := range all {
		if s.DisableModelInvocation || strings.TrimSpace(s.Path) == "" {
			continue
		}
		paths = append(paths, s.Path)
	}
	if len(paths) == 0 {
		return nil
	}
	_, err = c.rag.Index(ctx, rag.IndexRequest{
		Collection:  c.Collection(),
		SourcePaths: paths,
	})
	return err
}

// Search ranks skills for a user message using BM25 and optional RAG.
func (c *Catalog) Search(ctx context.Context, query string) ([]Match, error) {
	if c == nil || !c.Enabled() {
		return nil, nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	all, err := c.loader.LoadAll()
	if err != nil {
		return nil, err
	}
	topK := c.cfg.TopK
	if topK <= 0 {
		topK = 5
	}
	minScore := c.cfg.MinScore
	if minScore <= 0 {
		minScore = 0.55
	}

	byName := map[string]Skill{}
	for _, s := range all {
		if s.DisableModelInvocation {
			continue
		}
		byName[s.Name] = s
	}

	merged := map[string]Match{}
	for _, hit := range searchSkillsBM25(all, query) {
		if hit.Score < minScore || hit.Fit == "none" {
			continue
		}
		merged[hit.Name] = hit
	}

	if c.rag != nil {
		hits, err := c.rag.Query(ctx, rag.QueryRequest{
			Collection: c.Collection(),
			Query:      query,
			TopK:       topK * 2,
			MinScore:   minScore,
		})
		if err == nil {
			for _, hit := range hits {
				skill, ok := skillFromHit(byName, hit.Source)
				if !ok {
					continue
				}
				name := skill.Name
				prev, exists := merged[name]
				if !exists {
					merged[name] = Match{
						Name:        name,
						Description: skill.Description,
						Score:       hit.Score,
						MatchSource: "rag",
						Fit:         assessSkillFit(hit.Score, minScore),
					}
					continue
				}
				if hit.Score > prev.Score {
					prev.Score = hit.Score
				}
				switch prev.MatchSource {
				case "bm25":
					prev.MatchSource = "bm25+rag"
				case "rag":
					prev.MatchSource = "rag"
				default:
					prev.MatchSource = "bm25+rag"
				}
				prev.Fit = assessSkillFit(prev.Score, minScore)
				merged[name] = prev
			}
		}
	}

	out := make([]Match, 0, len(merged))
	for _, m := range merged {
		if m.Fit == "none" {
			continue
		}
		out = append(out, m)
	}
	sortSkillMatches(out)
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

func skillFromHit(byName map[string]Skill, source string) (Skill, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return Skill{}, false
	}
	for _, s := range byName {
		if s.Path == source {
			return s, true
		}
	}
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	for _, s := range byName {
		if s.Name == base {
			return s, true
		}
		if strings.HasSuffix(s.Path, source) {
			return s, true
		}
	}
	return Skill{}, false
}
