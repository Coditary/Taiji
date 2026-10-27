package memory

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

func TestServiceNilSafeAccessors(t *testing.T) {
	var svc *Service
	if svc.Enabled() || svc.AutoRecallOnError() || svc.KindCatalog() != nil {
		t.Fatal("nil service")
	}
	if svc.Collection() != "memory" || svc.MinRecallScore() != 0.72 || svc.DescribeDirs() != "" {
		t.Fatal("nil defaults")
	}
	if _, err := svc.Recall(context.Background(), RecallRequest{Query: "x"}); err == nil {
		t.Fatal("nil recall")
	}
}

func TestServiceRememberAndRecallDisabled(t *testing.T) {
	off := false
	cfg := &taijicfg.Config{Root: t.TempDir(), Memory: taijicfg.MemoryConfig{Enabled: &off}}
	svc := NewService(cfg, nil)
	if svc.Enabled() {
		t.Fatal("disabled")
	}
	if _, err := svc.Recall(context.Background(), RecallRequest{Query: "x"}); err != errDisabled {
		t.Fatalf("err=%v", err)
	}
}

func TestServiceDescribeDirsAndRAGRecall(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:   root,
		Memory: taijicfg.DefaultMemoryConfig(),
	}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	cfg.Memory.RAGCollection = "mem-col"
	cfg.Memory.MinRecallScore = 0.5
	ragSvc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	ragSvc.Backend = &recallRAGStub{}
	svc := NewService(cfg, ragSvc)
	desc := svc.DescribeDirs()
	if desc == "" || !strings.Contains(desc, "mem-col") {
		t.Fatalf("desc=%q", desc)
	}
	if svc.Collection() != "mem-col" || svc.MinRecallScore() != 0.5 {
		t.Fatal("accessors")
	}
	ctx := context.Background()
	_, err = svc.Remember(ctx, RememberRequest{
		Kind: string(KindPreference), Title: "t", RecallText: "alpha beta gamma", Scope: "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Recall(ctx, RecallRequest{Query: "beta", TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected recall hits")
	}
}

type recallRAGStub struct{}

func (recallRAGStub) Index(context.Context, string, rag.IndexRequest) (*rag.IndexResult, error) {
	return &rag.IndexResult{}, nil
}

func (recallRAGStub) Query(context.Context, string, rag.QueryRequest) ([]rag.Hit, error) {
	return []rag.Hit{{Text: "rag hit", Score: 0.95, Source: "mem.json"}}, nil
}
