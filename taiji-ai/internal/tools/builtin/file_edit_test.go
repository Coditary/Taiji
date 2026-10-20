package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyFileEditsSingleReplace(t *testing.T) {
	out, lines, err := applyFileEdits("alpha\nbeta\ngamma\n", []fileEdit{
		{OldText: "beta", NewText: "BETA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "BETA") || strings.Contains(out, "beta") {
		t.Fatalf("unexpected content: %q", out)
	}
	if len(lines) != 1 || lines[0] != 2 {
		t.Fatalf("lines=%v", lines)
	}
}

func TestApplyFileEditsRejectsAmbiguous(t *testing.T) {
	_, _, err := applyFileEdits("foo\nfoo\n", []fileEdit{{OldText: "foo", NewText: "bar"}})
	if err == nil || !strings.Contains(err.Error(), "2 times") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func TestApplyFileEditsWhitespaceTolerant(t *testing.T) {
	out, _, err := applyFileEdits("  hello  \nworld\n", []fileEdit{
		{OldText: "hello\nworld", NewText: "hi\nthere"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hi") || !strings.Contains(out, "there") {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestEditFileTool(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := (EditFile{}).RunWithMetadata(t.Context(), map[string]any{
		"path": path,
		"edits": []any{
			map[string]any{"old_text": "two", "new_text": "TWO"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "edited") {
		t.Fatalf("text=%q", res.Text)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "one\nTWO\nthree\n" {
		t.Fatalf("file=%q", raw)
	}
}

func TestReadFileLineNumbersAndWindow(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "big.txt")
	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, "line"+strings.Repeat("x", i))
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (ReadFile{}).Run(t.Context(), map[string]any{
		"path":       path,
		"start_line": 3,
		"end_line":   5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "   3 |") || !strings.Contains(out, "   5 |") {
		t.Fatalf("missing line numbers: %q", out)
	}
	if strings.Contains(out, "   6 |") {
		t.Fatalf("should not include line 6: %q", out)
	}
}

func TestReadFileLargeDefaultWindow(t *testing.T) {
	var lines []string
	for i := 1; i <= 350; i++ {
		lines = append(lines, fmt.Sprintf("L%d", i))
	}
	out, total, truncated := formatFileContentWithLineNumbers(strings.Join(lines, "\n"), 0, 0)
	if total != 350 || !truncated {
		t.Fatalf("total=%d truncated=%v", total, truncated)
	}
	if !strings.Contains(out, "   1 |") || !strings.Contains(out, " 200 |") {
		t.Fatalf("expected first 200 lines: %q", out[:200])
	}
	if strings.Contains(out, " 201 |") {
		t.Fatal("should truncate at 200")
	}
}
