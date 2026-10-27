package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobSearchValidationAndEmptyResults(t *testing.T) {
	_, err := (GlobSearch{}).Run(t.Context(), map[string]any{"pattern": ""})
	if err == nil || !strings.Contains(err.Error(), "pattern") {
		t.Fatalf("err=%v", err)
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "only.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = (GlobSearch{}).Run(t.Context(), map[string]any{"pattern": "*.go", "path": file})
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("err=%v", err)
	}

	if resolveTaijiToolsBinary() == "" {
		t.Skip("taiji-tools not built")
	}
	out, err := (GlobSearch{}).Run(t.Context(), map[string]any{
		"pattern": "*.missing",
		"path":    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no files matching") {
		t.Fatalf("out=%q", out)
	}
}
