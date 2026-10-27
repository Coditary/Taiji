package builtin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := (ListDir{}).Run(context.Background(), map[string]any{"path": root})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a.txt (file") {
		t.Fatalf("expected a.txt in listing: %q", out)
	}
	if !strings.Contains(out, "sub/ (dir)") {
		t.Fatalf("expected sub/ in listing: %q", out)
	}
}

func TestGlobSearch(t *testing.T) {
	if resolveTaijiToolsBinary() == "" {
		t.Skip("taiji-tools not built; run make taiji-tools in taiji-ai")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := (GlobSearch{}).Run(context.Background(), map[string]any{
		"path":    root,
		"pattern": "*.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "main.go") {
		t.Fatalf("expected main.go: %q", out)
	}
	if strings.Contains(out, "notes.txt") {
		t.Fatalf("did not expect notes.txt: %q", out)
	}
}

func TestGrepBuiltin(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not on PATH")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("func HelloWorld() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := (Grep{}).Run(context.Background(), map[string]any{
		"path":    root,
		"pattern": "HelloWorld",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "HelloWorld") {
		t.Fatalf("expected match: %q", out)
	}
}

func TestGrepFilesWithMatches(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not on PATH")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "miss.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (Grep{}).Run(context.Background(), map[string]any{
		"path":        root,
		"pattern":     "package",
		"output_mode": "files_with_matches",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hit.go") || strings.Contains(out, "miss.txt") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestExplorePathExpandsHome(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}
	abs, err := explorePath("~/")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(home)
	if err != nil {
		t.Fatal(err)
	}
	if abs != want {
		t.Fatalf("explorePath(~/) = %q, want %q", abs, want)
	}
}

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		rel, pattern string
		want         bool
	}{
		{"internal/tools/read_file.go", "**/*.go", true},
		{"README.md", "*.go", false},
		{"main.go", "*.go", true},
	}
	for _, tc := range cases {
		got, err := matchGlob(tc.rel, tc.pattern)
		if err != nil {
			t.Fatalf("matchGlob(%q, %q): %v", tc.rel, tc.pattern, err)
		}
		if got != tc.want {
			t.Fatalf("matchGlob(%q, %q) = %v, want %v", tc.rel, tc.pattern, got, tc.want)
		}
	}
}
