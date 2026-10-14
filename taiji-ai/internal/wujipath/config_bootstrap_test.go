package wujipath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEnsureWujiConfigCreatesOllamaDefaults(t *testing.T) {
	root := t.TempDir()
	modelsDir := filepath.Join(root, "models")
	if err := os.MkdirAll(filepath.Join(modelsDir, "ollama"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "ollama", "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)

	if err := EnsureWujiConfig(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".wuji", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg wujiFileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Drivers.Llama == nil || cfg.Drivers.Llama.UseOllamaAPI == nil || !*cfg.Drivers.Llama.UseOllamaAPI {
		t.Fatalf("expected use_ollama_api true, got %#v", cfg.Drivers.Llama)
	}
	if cfg.Drivers.Llama.OllamaThink == nil || *cfg.Drivers.Llama.OllamaThink {
		t.Fatalf("expected ollama_think false, got %#v", cfg.Drivers.Llama)
	}
}

func TestEnsureWujiConfigUpgradesDriverBinary(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	pluginBin := filepath.Join(home, "Dev", "Coditary", "plugins", "wuji", "llama", "bin")
	if err := os.MkdirAll(pluginBin, 0o755); err != nil {
		t.Fatal(err)
	}
	newDriver := filepath.Join(pluginBin, "wuji-driver-llama")
	if err := os.WriteFile(newDriver, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	oldDriver := filepath.Join(home, "bin", "wuji-driver-llama")
	if err := os.MkdirAll(filepath.Dir(oldDriver), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldDriver, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(home, ".wuji")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "drivers:\n  llama:\n    use_ollama_api: true\n    driver_bin: " + oldDriver + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if err := EnsureWujiConfig(home); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), newDriver) {
		t.Fatalf("expected upgraded driver_bin, got %s", data)
	}
}

func TestEnsureWujiConfigDoesNotOverwriteExisting(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".wuji")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "drivers:\n  llama:\n    use_ollama_api: false\n    models_dir: /custom/models\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureWujiConfig(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "use_ollama_api: true") {
		t.Fatalf("should not overwrite explicit use_ollama_api: %s", data)
	}
}
