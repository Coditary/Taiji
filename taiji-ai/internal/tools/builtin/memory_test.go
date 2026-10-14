package builtin

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/tools"
)

func TestMemoryRememberAcceptsQueryAliasForPersona(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:   root,
		Memory: taijicfg.DefaultMemoryConfig(),
	}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	svc := memory.NewService(cfg, nil)
	reg := tools.NewRegistry()
	EnableMemoryOrchestration(reg, svc)

	out, err := reg.Run(context.Background(), "memory_remember", `{
		"kind": "persona",
		"query": "Der Assistent heißt nun Luma."
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.Text == "" || !strings.Contains(out.Text, "remembered id=") {
		t.Fatalf("unexpected output: %q", out.Text)
	}
}

func TestMemoryRememberAcceptsTextAlias(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:   root,
		Memory: taijicfg.DefaultMemoryConfig(),
	}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	svc := memory.NewService(cfg, nil)
	reg := tools.NewRegistry()
	EnableMemoryOrchestration(reg, svc)

	out, err := reg.Run(context.Background(), "memory_remember", `{
		"kind": "preference",
		"title": "User name is Leonard",
		"text": "Der User heißt Leonard.",
		"recall_queries": ["wie heißt der user?", "wer bin ich?"]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.Text == "" || !strings.Contains(out.Text, "remembered id=") {
		t.Fatalf("unexpected output: %q", out.Text)
	}
}
