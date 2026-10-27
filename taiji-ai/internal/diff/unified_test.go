package diff

import (
	"strings"
	"testing"
)

func TestUnifiedNewFile(t *testing.T) {
	out := Unified("foo.go", "", "package main\n", 20)
	if !strings.Contains(out, "+++ b/foo.go") {
		t.Fatalf("expected header: %q", out)
	}
	if !strings.Contains(out, "+package main") {
		t.Fatalf("expected added line: %q", out)
	}
}

func TestUnifiedModified(t *testing.T) {
	old := "a\nb\nc\n"
	newText := "a\nB\nc\n"
	out := Unified("x.txt", old, newText, 20)
	if !strings.Contains(out, "-b") || !strings.Contains(out, "+B") {
		t.Fatalf("expected line change: %q", out)
	}
}

func TestUnifiedIncludesContextWindow(t *testing.T) {
	old := "l1\nl2\nl3\nl4\nl5\nl6\nl7\n"
	newText := "l1\nl2\nl3\nL4\nl5\nl6\nl7\n"
	out := Unified("x.txt", old, newText, 50)
	if !strings.Contains(out, " l3") || !strings.Contains(out, " l5") {
		t.Fatalf("expected context lines around change: %q", out)
	}
}

func TestUnifiedTruncate(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "line"
	}
	out := Unified("big.txt", "", strings.Join(lines, "\n"), 5)
	if !strings.Contains(out, "more lines") {
		t.Fatalf("expected truncation marker: %q", out)
	}
}
