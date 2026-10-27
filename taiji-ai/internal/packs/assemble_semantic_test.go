package packs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

type ragStub struct {
	hits []rag.Hit
}

func (s *ragStub) Index(context.Context, string, rag.IndexRequest) (*rag.IndexResult, error) {
	return &rag.IndexResult{}, nil
}

func (s *ragStub) Query(context.Context, string, rag.QueryRequest) ([]rag.Hit, error) {
	return s.hits, nil
}

func TestAssembleExplicitMentionAndManual(t *testing.T) {
	root := t.TempDir()
	packRoot := filepath.Join(root, ".taiji", "packs", "lore")
	_ = os.MkdirAll(packRoot, 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "tavern.md"), []byte(`---
name: tavern
injection: keyword
keywords: [inn]
---
Tavern lore.
`), 0o644)
	_ = os.WriteFile(filepath.Join(packRoot, "manual.md"), []byte(`---
injection: manual
description: On demand lore
---
Hidden body.
`), 0o644)

	cfg := &taijicfg.Config{Root: root, Packs: taijicfg.PacksConfig{Dirs: []string{filepath.Join(root, ".taiji", "packs")}}}
	idx := NewIndex(NewLoader(cfg.Packs.Dirs), nil, cfg, root)
	if err := idx.Reload(); err != nil {
		t.Fatal(err)
	}
	res, err := idx.Assemble(context.Background(), AssembleRequest{
		Goal: "[Context: tavern] hello @tavern",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.UserPreface == "" {
		t.Fatalf("expected explicit user preface: %#v", res)
	}
	if !strings.Contains(res.SystemAppendix, "manual") {
		t.Fatalf("manual catalog missing: %q", res.SystemAppendix)
	}
	summary := FormatInjectSummary(res.Lines)
	if summary == "" || !strings.Contains(summary, "Injected") {
		t.Fatalf("summary=%q", summary)
	}
}

func TestAssembleSemanticRAG(t *testing.T) {
	root := t.TempDir()
	packRoot := filepath.Join(root, ".taiji", "packs", "lore")
	_ = os.MkdirAll(packRoot, 0o755)
	path := filepath.Join(packRoot, "deep.md")
	_ = os.WriteFile(path, []byte(`---
injection: semantic
name: deep
---
Semantic body.
`), 0o644)

	cfg := &taijicfg.Config{Root: root, Packs: taijicfg.PacksConfig{Dirs: []string{filepath.Join(root, ".taiji", "packs")}}}
	svc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	svc.Backend = &ragStub{hits: []rag.Hit{{Text: "chunk text", Score: 0.9, Source: path}}}
	idx := NewIndex(NewLoader(cfg.Packs.Dirs), svc, cfg, root)
	if err := idx.Reload(); err != nil {
		t.Fatal(err)
	}
	res, err := idx.Assemble(context.Background(), AssembleRequest{Goal: "find lore"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.SystemAppendix, "semantic") || !strings.Contains(res.SystemAppendix, "chunk text") {
		t.Fatalf("semantic inject missing: %q", res.SystemAppendix)
	}
}
