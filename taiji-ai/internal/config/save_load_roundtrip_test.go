package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSaveAndLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	taijiDir := filepath.Join(root, dirName)
	if err := os.MkdirAll(taijiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := defaultConfig(root)
	cfg.StorageLocal = true
	cfg.DefaultProvider = "dummy_a"
	cfg.Providers = map[string]ProviderConfig{
		"dummy_a": {Type: "dummy", Model: "alpha"},
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadAt(root, LoadOptions{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultProvider != "dummy_a" || loaded.Providers["dummy_a"].Model != "alpha" {
		t.Fatalf("loaded=%+v", loaded)
	}
	if RuntimeDir(root) != taijiDir {
		t.Fatalf("runtime=%s", RuntimeDir(root))
	}
}

func TestFindProjectRootWithGoMod(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := findProjectRoot(nested)
	if !ok || got != root {
		t.Fatalf("root=%q ok=%v", got, ok)
	}
}
