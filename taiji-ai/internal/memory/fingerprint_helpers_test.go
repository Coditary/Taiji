package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUniquePathAndFileExists(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "note.md")
	if err := os.WriteFile(base, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := UniquePath(dir, "note.md")
	if got == base {
		t.Fatalf("expected unique suffix, got %q", got)
	}
	if !fileExists(base) || fileExists(filepath.Join(dir, "missing")) {
		t.Fatal("fileExists")
	}
	if itoa(42) != "42" {
		t.Fatal("itoa")
	}
}
