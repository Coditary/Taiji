package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

type stubRAGBackend struct {
	indexFn func(ctx context.Context, storeRoot string, req rag.IndexRequest) (*rag.IndexResult, error)
	queryFn func(ctx context.Context, storeRoot string, req rag.QueryRequest) ([]rag.Hit, error)
}

func (s *stubRAGBackend) Index(ctx context.Context, storeRoot string, req rag.IndexRequest) (*rag.IndexResult, error) {
	if s.indexFn != nil {
		return s.indexFn(ctx, storeRoot, req)
	}
	return &rag.IndexResult{
		Collection:  req.Collection,
		Message:     "indexed",
		ChunksAdded: 2,
		TotalChunks: 2,
	}, nil
}

func (s *stubRAGBackend) Query(ctx context.Context, storeRoot string, req rag.QueryRequest) ([]rag.Hit, error) {
	if s.queryFn != nil {
		return s.queryFn(ctx, storeRoot, req)
	}
	return []rag.Hit{{
		Scope:  "local",
		Score:  0.91,
		Text:   "mock hit text",
		Source: "doc.md",
	}}, nil
}

func testRAGService(t *testing.T) *rag.Service {
	t.Helper()
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{WujiRoot: t.TempDir()}}
	svc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	svc.Backend = &stubRAGBackend{}
	if svc.DefaultCollection == "" {
		svc.DefaultCollection = "default"
	}
	return svc
}

func TestRAGSearchRun(t *testing.T) {
	svc := testRAGService(t)
	out, err := (RAGSearch{svc: svc}).Run(context.Background(), map[string]any{
		"query": "hello",
		"top_k": float64(3),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "mock hit") || !strings.Contains(out, "scope:") {
		t.Fatalf("out=%q", out)
	}
}

func TestRAGSearchNoHits(t *testing.T) {
	svc := testRAGService(t)
	svc.Backend = &stubRAGBackend{
		queryFn: func(ctx context.Context, storeRoot string, req rag.QueryRequest) ([]rag.Hit, error) {
			return nil, nil
		},
	}
	out, err := (RAGSearch{svc: svc}).Run(context.Background(), map[string]any{"query": "empty"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no hits") {
		t.Fatalf("out=%q", out)
	}
}

func TestRAGIndexRun(t *testing.T) {
	svc := testRAGService(t)
	root := t.TempDir()
	doc := filepath.Join(root, "note.md")
	if err := os.WriteFile(doc, []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (RAGIndex{svc: svc}).Run(context.Background(), map[string]any{
		"paths": []any{doc},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "indexed") && !strings.Contains(out, "chunks") {
		t.Fatalf("out=%q", out)
	}
}

func TestRAGIndexRequiresPaths(t *testing.T) {
	svc := testRAGService(t)
	_, err := (RAGIndex{svc: svc}).Run(context.Background(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "paths") {
		t.Fatalf("err=%v", err)
	}
}

func TestIndexingWriteFileAutoIndexes(t *testing.T) {
	svc := testRAGService(t)
	root := t.TempDir()
	path := filepath.Join(root, "fresh.md")
	tool := NewIndexingWriteFile(svc)
	out, err := tool.Run(context.Background(), map[string]any{
		"path":    path,
		"content": "hello rag",
	})
	if err != nil || !strings.Contains(out, "auto-indexed") {
		t.Fatalf("err=%v out=%q", err, out)
	}
}

func TestIndexingEditFileAutoIndexes(t *testing.T) {
	svc := testRAGService(t)
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tool := NewIndexingEditFile(svc)
	out, err := tool.Run(context.Background(), map[string]any{
		"path": path,
		"edits": []any{
			map[string]any{"old_text": "v1", "new_text": "v2"},
		},
	})
	if err != nil || !strings.Contains(out, "edited") || !strings.Contains(out, "auto-indexed") {
		t.Fatalf("err=%v out=%q", err, out)
	}
}
