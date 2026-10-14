package memory

import (
	"context"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestRecallUsesBM25BeforeRAG(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:   root,
		Memory: taijicfg.DefaultMemoryConfig(),
	}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	svc := NewService(cfg, nil)
	ctx := context.Background()

	_, err := svc.Remember(ctx, RememberRequest{
		Kind:        string(KindPreference),
		Title:       "Name des Benutzers",
		RecallText:  "Der Benutzer heißt Leonard.",
		VerifiedFix: "Leonard",
		Scope:       "local",
	})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := svc.Recall(ctx, RecallRequest{
		Query: "Wie heiße ich?",
		Kind:  string(KindPreference),
		TopK:  5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected BM25 recall hit for name question")
	}
	if entries[0].MatchSource != "bm25" {
		t.Fatalf("expected bm25 source, got %q", entries[0].MatchSource)
	}
}
