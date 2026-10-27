package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryKindsSearchPaths(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := t.TempDir()
	projectFile := filepath.Join(RuntimeDir(root), memoryKindsFile)
	if err := os.MkdirAll(filepath.Dir(projectFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectFile, []byte("kinds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := MemoryKindsSearchPaths(&Config{Root: root})
	if len(paths) == 0 {
		t.Fatal("expected at least one kinds path")
	}
	last := paths[len(paths)-1]
	if last != projectFile {
		t.Fatalf("expected project file last, got %v", paths)
	}
}

func TestBundledMemoryKindsPath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	var found string
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "taiji-ai", "global", "memory-kinds.yaml")
		if _, err := os.Stat(candidate); err == nil {
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
		t.Skip("bundled memory-kinds.yaml not found from test cwd")
	}
	if BundledMemoryKindsPath(dir) == "" {
		t.Fatal("expected bundled path")
	}
}
