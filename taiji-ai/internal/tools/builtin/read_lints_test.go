package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadLintsGoModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/linttest\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := ReadLints{}
	out, err := tool.Run(context.Background(), map[string]any{"paths": []any{dir}})
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected lint output section")
	}
}
