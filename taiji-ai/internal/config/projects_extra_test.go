package config

import (
	"path/filepath"
	"testing"
)

func TestRegisterProjectUpdatesDisplayName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := filepath.Join(t.TempDir(), "my-project")
	if err := RegisterProject(root, false); err != nil {
		t.Fatal(err)
	}
	entries, err := loadProjectsRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries=%d", len(entries))
	}
	if entries[0].Name == "" {
		t.Fatal("expected display name")
	}
	if cleanProjectRoot("  "+root+"  ") != filepath.Clean(root) {
		t.Fatal("clean root")
	}
}
