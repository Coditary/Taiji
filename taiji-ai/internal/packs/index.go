package packs

import (
	"context"
	"strings"
	"sync"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

// Index holds in-memory pack nodes and optional Wuji RAG indexing for semantic nodes.
type Index struct {
	loader *Loader
	rag    *rag.Service
	cfg    taijicfg.PacksConfig
	workDir string

	mu    sync.RWMutex
	nodes []PackNode
}

// NewIndex builds a pack index.
func NewIndex(loader *Loader, ragSvc *rag.Service, cfg *taijicfg.Config, workDir string) *Index {
	if loader == nil || cfg == nil || !cfg.ResolvedPacks().EnabledOrDefault() {
		return nil
	}
	packCfg := cfg.ResolvedPacks()
	return &Index{
		loader:  loader,
		rag:     ragSvc,
		cfg:     packCfg,
		workDir: workDir,
	}
}

// Enabled reports whether pack assembly is active.
func (idx *Index) Enabled() bool {
	return idx != nil
}

// Reload scans pack directories and legacy AGENTS.md for the index work dir.
func (idx *Index) Reload() error {
	if idx == nil {
		return nil
	}
	return idx.ReloadAt(idx.workDir)
}

// ReloadAt rescans packs and AGENTS.md for the given project/workspace root.
func (idx *Index) ReloadAt(workDir string) error {
	if idx == nil {
		return nil
	}
	workDir = strings.TrimSpace(workDir)
	if workDir != "" {
		idx.workDir = workDir
	}
	packs, legacy, err := idx.loader.LoadAll(idx.workDir)
	if err != nil {
		return err
	}
	nodes := FlattenNodes(packs, legacy)
	idx.mu.Lock()
	idx.nodes = nodes
	idx.mu.Unlock()
	return nil
}

// Nodes returns a snapshot of indexed nodes.
func (idx *Index) Nodes() []PackNode {
	if idx == nil {
		return nil
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return append([]PackNode(nil), idx.nodes...)
}

// Collection returns the Wuji RAG collection name for semantic pack nodes.
func (idx *Index) Collection() string {
	if idx == nil || strings.TrimSpace(idx.cfg.RAGCollection) == "" {
		return "context-packs"
	}
	return strings.TrimSpace(idx.cfg.RAGCollection)
}

// EnsureSemanticIndexed indexes semantic-mode pack files into Wuji.
func (idx *Index) EnsureSemanticIndexed(ctx context.Context) error {
	if idx == nil || idx.rag == nil {
		return nil
	}
	paths := make([]string, 0)
	for _, n := range idx.Nodes() {
		if n.EffectiveInjection() != ModeSemantic {
			continue
		}
		if path := strings.TrimSpace(n.Path); path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	_, err := idx.rag.Index(ctx, rag.IndexRequest{
		Collection:  idx.Collection(),
		SourcePaths: paths,
	})
	return err
}

