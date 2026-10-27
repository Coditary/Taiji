package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeDirManagedByDefault(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := filepath.Join(t.TempDir(), "my-repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dataHome, "taiji", "projects", ProjectKey(root))
	if got := RuntimeDir(root); got != want {
		t.Fatalf("RuntimeDir = %q, want %q", got, want)
	}
}

func TestRuntimeDirLocalWhenRegistered(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	root := filepath.Join(t.TempDir(), "portable")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RegisterProject(root, true); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, dirName)
	if got := RuntimeDir(root); got != want {
		t.Fatalf("RuntimeDir = %q, want %q", got, want)
	}
}

func TestSaveUsesManagedStoreWithoutDotTaijiInProject(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := defaultConfig(root)
	cfg.DefaultProvider = "test"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, dirName)); err == nil {
		t.Fatal("expected no .taiji in project root")
	}
	managed := filepath.Join(RuntimeDir(root), configFile)
	if _, err := os.Stat(managed); err != nil {
		t.Fatalf("managed config missing: %v", err)
	}
}
