package rag_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

type recordingBackend struct {
	lastPaths []string
}

func (r *recordingBackend) Index(ctx context.Context, storeRoot string, req rag.IndexRequest) (*rag.IndexResult, error) {
	r.lastPaths = append([]string(nil), req.SourcePaths...)
	return &rag.IndexResult{
		Collection:  req.Collection,
		Scope:       "local",
		Message:     "ok",
		ChunksAdded: 1,
		TotalChunks: 1,
	}, nil
}

func (r *recordingBackend) Query(ctx context.Context, storeRoot string, req rag.QueryRequest) ([]rag.Hit, error) {
	return nil, nil
}

func TestAutoIndexFileMarkdown(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{WujiRoot: t.TempDir()}}
	svc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	backend := &recordingBackend{}
	svc.Backend = backend
	path := filepath.Join(cfg.Root, "docs", "readme.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# doc"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := rag.AutoIndexFile(context.Background(), svc, path)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(backend.lastPaths) != 1 || backend.lastPaths[0] != path {
		t.Fatalf("res=%+v paths=%v", res, backend.lastPaths)
	}
	note := rag.FormatAutoIndexNote(res, nil)
	if !strings.Contains(note, "auto-indexed") {
		t.Fatalf("note=%q", note)
	}
}

func TestFormatAutoIndexNoteError(t *testing.T) {
	note := rag.FormatAutoIndexNote(nil, os.ErrNotExist)
	if !strings.Contains(note, "skipped") {
		t.Fatalf("note=%q", note)
	}
}

func TestAutoIndexPackMarkdownUsesCollection(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{WujiRoot: t.TempDir()}}
	svc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	backend := &recordingBackend{}
	svc.Backend = backend
	path := filepath.Join(cfg.Root, ".taiji", "packs", "demo", "node.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\nname: n\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := rag.AutoIndexFile(context.Background(), svc, path)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Collection != "context-packs" {
		t.Fatalf("res=%+v", res)
	}
}
