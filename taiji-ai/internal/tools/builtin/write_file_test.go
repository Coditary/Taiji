package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileDiffMetadata(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "demo.txt")
	tool := WriteFile{}

	first, err := tool.RunWithMetadata(context.Background(), map[string]any{
		"path":    path,
		"content": "hello\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Data["is_new"] != true {
		t.Fatalf("expected new file metadata: %#v", first.Data)
	}
	diff, _ := first.Data["diff"].(string)
	if !strings.Contains(diff, "+hello") {
		t.Fatalf("expected added line in diff: %q", diff)
	}

	second, err := tool.RunWithMetadata(context.Background(), map[string]any{
		"path":    path,
		"content": "hello\nworld\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	diff, _ = second.Data["diff"].(string)
	if !strings.Contains(diff, "+world") {
		t.Fatalf("expected appended line in diff: %q", diff)
	}
}

func TestWriteFileRunCompatibility(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "x.txt")
	out, err := (WriteFile{}).Run(context.Background(), map[string]any{
		"path":    path,
		"content": "ok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "wrote") {
		t.Fatalf("unexpected output: %q", out)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "ok" {
		t.Fatalf("unexpected file content: %q", raw)
	}
}
