package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestLoadKindCatalogIncludesPersona(t *testing.T) {
	catalog := LoadKindCatalog(&taijicfg.Config{Root: findRepoRoot(t)})
	if _, ok := catalog.Lookup("persona"); !ok {
		t.Fatal("expected persona kind in catalog")
	}
	if !strings.Contains(catalog.BuildCatalogAppendix(), "persona") {
		t.Fatalf("appendix missing persona: %s", catalog.BuildCatalogAppendix())
	}
}

func TestKindCatalogProjectOverride(t *testing.T) {
	root := t.TempDir()
	projectFile := filepath.Join(taijicfg.RuntimeDir(root), "memory-kinds.yaml")
	if err := os.MkdirAll(filepath.Dir(projectFile), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `kinds:
  - name: client
    description: Facts about a client account
    store_when: user mentions client-specific constraints
    recall_when: working on that client's project
    requires_verified_fix: false
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog := LoadKindCatalog(&taijicfg.Config{Root: root})
	if _, ok := catalog.Lookup("client"); !ok {
		t.Fatal("expected custom client kind")
	}
	if _, ok := catalog.Lookup("preference"); !ok {
		t.Fatal("expected bundled preference kind to remain")
	}
}

func TestKindCatalogInlineConfigOverride(t *testing.T) {
	cfg := &taijicfg.Config{
		Root: findRepoRoot(t),
		Memory: func() taijicfg.MemoryConfig {
			m := taijicfg.DefaultMemoryConfig()
			m.Kinds = []taijicfg.MemoryKindSpec{
				{
					Name:        "client",
					Description: "Inline client facts",
					StoreWhen:   "client context",
					RecallWhen:  "client work",
				},
			}
			return m
		}(),
	}
	catalog := LoadKindCatalog(cfg)
	if def, ok := catalog.Lookup("client"); !ok || def.Description != "Inline client facts" {
		t.Fatalf("expected inline kind override, got %+v", def)
	}
}

func TestSuggestRecallKindsPersona(t *testing.T) {
	catalog := LoadKindCatalog(&taijicfg.Config{Root: findRepoRoot(t)})
	kinds := catalog.SuggestRecallKinds("wie heißt du?")
	found := false
	for _, kind := range kinds {
		if kind == KindPersona {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected persona for assistant name question, got %v", kinds)
	}
}

func TestSuggestStoreKindsPersona(t *testing.T) {
	catalog := LoadKindCatalog(&taijicfg.Config{Root: findRepoRoot(t)})
	kinds := catalog.SuggestStoreKinds("Ab jetzt heißt du Luma")
	found := false
	for _, kind := range kinds {
		if kind == KindPersona {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected persona for store hint, got %v", kinds)
	}
}

func TestParseKindCustomKind(t *testing.T) {
	root := t.TempDir()
	projectFile := filepath.Join(taijicfg.RuntimeDir(root), "memory-kinds.yaml")
	if err := os.MkdirAll(filepath.Dir(projectFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectFile, []byte("kinds:\n  - name: client\n    description: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog := LoadKindCatalog(&taijicfg.Config{Root: root})
	if _, err := catalog.ParseKind("client"); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.ParseKind("bogus"); err == nil {
		t.Fatal("expected unknown kind error")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "taiji-ai", "global", "memory-kinds.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return wd
}
