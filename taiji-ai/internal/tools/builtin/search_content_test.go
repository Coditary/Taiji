package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseContentSearchArgs(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "a.go")
	if err := os.WriteFile(f, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parsed, err := parseContentSearchArgs(map[string]any{
		"pattern": "package",
		"path":    root,
		"glob":    "*.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.root != root || parsed.pattern != "package" {
		t.Fatalf("parsed=%+v", parsed)
	}
	if _, err := parseContentSearchArgs(map[string]any{}); err == nil {
		t.Fatal("pattern required")
	}
	if _, err := parseContentSearchArgs(map[string]any{"pattern": "x", "output_mode": "nope"}); err == nil {
		t.Fatal("bad output mode")
	}
}

func TestRunContentSearchFindsMatch(t *testing.T) {
	if _, err := requireRipgrep(); err != nil {
		t.Skip("rg not available")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "needle.txt"), []byte("hello needle world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := RunContentSearch(context.Background(), map[string]any{
		"pattern": "needle",
		"path":    root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "needle") {
		t.Fatalf("out=%q", out)
	}
	out, err = RunContentSearch(context.Background(), map[string]any{
		"pattern":     "missing",
		"path":        root,
		"output_mode": "files_with_matches",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no matches") {
		t.Fatalf("out=%q", out)
	}
}
