package session

import (
	"path/filepath"
	"testing"
)

func TestProjectRootFromDir(t *testing.T) {
	root := filepath.Clean("/tmp/taiji-project")
	dir := filepath.Join(root, ".taiji", "sessions", "12345")
	got := ProjectRoot(&Session{Dir: dir})
	if got != root {
		t.Fatalf("ProjectRoot = %q, want %q", got, root)
	}
}
