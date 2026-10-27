package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileLineRange(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	path := filepath.Join(root, "lines.txt")
	content := "one\ntwo\nthree\nfour\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (ReadFile{}).Run(ctx, map[string]any{
		"path":       path,
		"start_line": 2,
		"end_line":   3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "two") || !strings.Contains(out, "three") || strings.Contains(out, "four") {
		t.Fatalf("out=%q", out)
	}
}

func TestReadFileRequiresPath(t *testing.T) {
	_, ctx := useProjectWorkspace(t)
	_, err := (ReadFile{}).Run(ctx, map[string]any{})
	if err == nil {
		t.Fatal("expected error")
	}
}
