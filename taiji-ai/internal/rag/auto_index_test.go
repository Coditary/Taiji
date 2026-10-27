package rag

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestAutoIndexFileAndFormatNote(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{WujiRoot: t.TempDir()}}
	svc, err := NewService(cfg, Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	svc.Backend = &stubAutoIndexBackend{}
	path := filepath.Join(cfg.Root, "docs", "readme.md")
	res, err := AutoIndexFile(context.Background(), svc, path)
	if err != nil || res == nil || res.ChunksAdded != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	note := FormatAutoIndexNote(res, nil)
	if !strings.Contains(note, "indexed") {
		t.Fatalf("note=%q", note)
	}
	if note := FormatAutoIndexNote(nil, errIndexFail{}); !strings.Contains(note, "skipped") {
		t.Fatalf("error note=%q", note)
	}
	if _, err := AutoIndexFile(context.Background(), nil, path); err == nil {
		t.Fatal("nil service")
	}
	if res, err := AutoIndexFile(context.Background(), svc, "image.png"); err != nil || res != nil {
		t.Fatalf("skip png: res=%v err=%v", res, err)
	}
}

type errIndexFail struct{}

func (errIndexFail) Error() string { return "boom" }

type stubAutoIndexBackend struct{}

func (stubAutoIndexBackend) Index(_ context.Context, _ string, req IndexRequest) (*IndexResult, error) {
	return &IndexResult{Collection: req.Collection, ChunksAdded: 1, Message: "indexed"}, nil
}

func (stubAutoIndexBackend) Query(context.Context, string, QueryRequest) ([]Hit, error) {
	return nil, nil
}
