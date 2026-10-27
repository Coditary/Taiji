package config

import (
	"os"
	"path/filepath"
	"testing"
)

// User saves keybinding_style=standard in a local .taiji project; global config still
// has vim from an earlier session. On restart, standard must not be overwritten.
func TestLoadLocalProjectKeybindingStyleNotOverriddenByGlobal(t *testing.T) {
	configHome := t.TempDir()
	dataHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)

	root := filepath.Join(t.TempDir(), "portable")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RegisterProject(root, true); err != nil {
		t.Fatal(err)
	}

	globalDir := GlobalConfigDir()
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	globalPath := filepath.Join(globalDir, configFile)
	globalYAML := `ui:
  keybinding_style: vim
`
	if err := os.WriteFile(globalPath, []byte(globalYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := defaultConfig(root)
	cfg.StorageLocal = true
	cfg.UI.KeybindingStyle = "standard"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadAt(root, LoadOptions{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.UI.KeybindingStyleOrDefault() != "standard" {
		t.Fatalf("keybinding_style = %q, want standard (global vim must not override local save)",
			loaded.UI.KeybindingStyleOrDefault())
	}
}

func TestSaveLocalProjectAlsoUpdatesGlobalKeybindingStyle(t *testing.T) {
	configHome := t.TempDir()
	dataHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)

	root := filepath.Join(t.TempDir(), "portable")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RegisterProject(root, true); err != nil {
		t.Fatal(err)
	}

	cfg := defaultConfig(root)
	cfg.StorageLocal = true
	cfg.UI.KeybindingStyle = "standard"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	globalPath := filepath.Join(GlobalConfigDir(), configFile)
	raw, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("global config: %v", err)
	}
	if !containsAll(string(raw), "keybinding_style", "standard") {
		t.Fatalf("expected global ui.keybinding_style standard, got:\n%s", string(raw))
	}
}
