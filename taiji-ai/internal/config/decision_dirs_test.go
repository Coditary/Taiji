package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecisionDirs(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	global := GlobalDecisionsDir()
	if global != filepath.Join(configHome, "taiji", decisionsSubdir) {
		t.Fatalf("global=%q", global)
	}
	root := t.TempDir()
	project := ProjectDecisionsDir(root)
	if project != filepath.Join(RuntimeDir(root), decisionsSubdir) {
		t.Fatalf("project=%q", project)
	}
	dirs := DefaultDecisionProfileDirs(root)
	if len(dirs) < 2 {
		t.Fatalf("dirs=%v", dirs)
	}
}

func TestBundledDecisionsDirFromProject(t *testing.T) {
	root := t.TempDir()
	bundled := filepath.Join(root, "global", decisionsSubdir)
	if err := os.MkdirAll(bundled, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := BundledDecisionsDir(root); got != bundled {
		t.Fatalf("got %q want %q", got, bundled)
	}
}
