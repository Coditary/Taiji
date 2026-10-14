package memory

import (
	"context"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestRememberPersonaWithoutTitle(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:   root,
		Memory: taijicfg.DefaultMemoryConfig(),
	}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	svc := NewService(cfg, nil)
	ctx := context.Background()

	entry, err := svc.Remember(ctx, RememberRequest{
		Kind:       string(KindPersona),
		RecallText: "Der Assistent heißt nun Luma.",
		Scope:      "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Title == "" {
		t.Fatal("expected derived title")
	}
	if entry.Kind != KindPersona {
		t.Fatalf("kind=%s", entry.Kind)
	}
}

func TestDeriveMemoryTitle(t *testing.T) {
	title := deriveMemoryTitle(KindPersona, "Der Assistent heißt nun Luma.")
	if title != "Der Assistent heißt nun Luma." {
		t.Fatalf("unexpected title: %q", title)
	}
}
