package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultSkillsDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data"))

	root := filepath.Join(home, "project")
	got := DefaultSkillsDirs(root)
	if len(got) != 2 {
		t.Fatalf("dirs = %v", got)
	}
	wantProject := filepath.Join(home, "xdg-data", "taiji", "projects", ProjectKey(root), "skills")
	wantGlobal := filepath.Join(home, "xdg-config", "taiji", "skills")
	if got[0] != wantProject {
		t.Fatalf("project dir = %q want %q", got[0], wantProject)
	}
	if got[1] != wantGlobal {
		t.Fatalf("global dir = %q want %q", got[1], wantGlobal)
	}
}

func TestEnsureSkillsDirsUsesDefaultsWhenUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg-config"))

	root := filepath.Join(home, "repo")
	cfg := &Config{Root: root}
	ensureSkillsDirs(cfg)
	if len(cfg.Skills.Dirs) != 2 {
		t.Fatalf("dirs = %v", cfg.Skills.Dirs)
	}
}

func TestEnsureSkillsDirsPreservesExplicitDirs(t *testing.T) {
	cfg := &Config{
		Root:   "/tmp",
		Skills: SkillsConfig{Dirs: []string{"/custom/skills"}},
	}
	ensureSkillsDirs(cfg)
	if len(cfg.Skills.Dirs) != 1 || cfg.Skills.Dirs[0] != "/custom/skills" {
		t.Fatalf("dirs = %v", cfg.Skills.Dirs)
	}
}
