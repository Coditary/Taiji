package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobSearchUsesGlobwalk(t *testing.T) {
	if resolveTaijiToolsBinary() == "" {
		t.Skip("taiji-tools not built; run make taiji-tools in taiji-ai")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (GlobSearch{}).Run(t.Context(), map[string]any{
		"pattern": "*.go",
		"path":    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "hello.go" {
			return
		}
	}
	t.Fatalf("expected hello.go in %q", out)
}
