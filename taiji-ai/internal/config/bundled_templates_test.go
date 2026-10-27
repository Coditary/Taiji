package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundledTemplatesDirFindsShippedTemplates(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// When tests run from taiji-ai/, walk up to monorepo root.
	dir := wd
	var found string
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "taiji-ai", "global", "templates")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			found = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if found == "" {
		t.Skip("bundled templates directory not found from test cwd")
	}
	got := BundledTemplatesDir("")
	if got == "" {
		t.Fatal("expected bundled templates dir")
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("stat %q: %v", got, err)
	}
}

func TestBundledTemplatesDirProjectRelative(t *testing.T) {
	root := t.TempDir()
	bundled := filepath.Join(root, "taiji-ai", "global", "templates")
	if err := os.MkdirAll(bundled, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := BundledTemplatesDir(root); got != bundled {
		t.Fatalf("got %q want %q", got, bundled)
	}
}
