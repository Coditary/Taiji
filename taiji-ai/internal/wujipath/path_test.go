package wujipath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasWujiStateDir(t *testing.T) {
	root := t.TempDir()
	if hasWujiStateDir(root) {
		t.Fatal("expected false without .wuji")
	}
	if err := os.Mkdir(filepath.Join(root, ".wuji"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !hasWujiStateDir(root) {
		t.Fatal("expected true with .wuji")
	}
	if hasWujiStateDir("") {
		t.Fatal("empty root should be false")
	}
}

func TestDiscoverInstalledCoreBinariesEmptyHome(t *testing.T) {
	// Ensures glob patterns run without panic when home exists.
	_ = discoverInstalledCoreBinaries()
}
