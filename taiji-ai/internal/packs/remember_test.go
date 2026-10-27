package packs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendToNodeCreatesAndAppends(t *testing.T) {
	root := t.TempDir()
	packRoot := filepath.Join(root, "packs")
	packName := "story"
	if err := os.MkdirAll(filepath.Join(packRoot, packName), 0o755); err != nil {
		t.Fatal(err)
	}

	path, err := AppendToNode([]string{packRoot}, packName, "lore/intro", "First fact.")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "First fact.") {
		t.Fatalf("content=%q", raw)
	}

	path2, err := AppendToNode([]string{packRoot}, packName, "lore/intro", "Second fact.")
	if err != nil {
		t.Fatal(err)
	}
	if path2 != path {
		t.Fatalf("paths differ: %q vs %q", path, path2)
	}
	raw, _ = os.ReadFile(path)
	if !strings.Contains(string(raw), "Second fact.") {
		t.Fatalf("append failed: %q", raw)
	}
}

func TestAppendToNodeRejectsInvalid(t *testing.T) {
	_, err := AppendToNode(nil, "", "node", "x")
	if err == nil {
		t.Fatal("expected validation error")
	}
	_, err = AppendToNode([]string{t.TempDir()}, "p", "../escape", "x")
	if err == nil {
		t.Fatal("expected path escape error")
	}
}
