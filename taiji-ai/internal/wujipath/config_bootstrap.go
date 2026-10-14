package wujipath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type wujiFileConfig struct {
	Drivers wujiDriversConfig `yaml:"drivers"`
}

type wujiDriversConfig struct {
	Llama *wujiLlamaConfig `yaml:"llama,omitempty"`
}

type wujiLlamaConfig struct {
	UseOllamaAPI  *bool  `yaml:"use_ollama_api,omitempty"`
	OllamaThink   *bool  `yaml:"ollama_think,omitempty"`
	OllamaAPI     string `yaml:"ollama_api,omitempty"`
	ModelsDir     string `yaml:"models_dir,omitempty"`
	DriverBin     string `yaml:"driver_bin,omitempty"`
	AutoStartDriver *bool `yaml:"auto_start_driver,omitempty"`
}

// EnsureWujiConfig creates ~/.wuji/config.yaml with Ollama-friendly llama driver defaults.
func EnsureWujiConfig(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		root = home
	}
	configDir := filepath.Join(root, ".wuji")
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	var existing wujiFileConfig
	if data, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	} else if !os.IsNotExist(err) {
		return err
	}

	llama := existing.Drivers.Llama
	if llama == nil {
		llama = &wujiLlamaConfig{}
	}
	changed := existing.Drivers.Llama == nil

	if llama.UseOllamaAPI == nil {
		useAPI := true
		llama.UseOllamaAPI = &useAPI
		changed = true
	}
	if llama.OllamaThink == nil {
		think := false
		llama.OllamaThink = &think
		changed = true
	}
	if strings.TrimSpace(llama.OllamaAPI) == "" {
		llama.OllamaAPI = "http://127.0.0.1:11434"
		changed = true
	}
	if strings.TrimSpace(llama.ModelsDir) == "" {
		if dir := discoverLlamaModelsDir(); dir != "" {
			llama.ModelsDir = dir
			changed = true
		}
	}
	if bin := discoverLlamaDriverBinary(); bin != "" && llama.DriverBin != bin {
		llama.DriverBin = bin
		changed = true
	}
	if llama.AutoStartDriver == nil {
		auto := true
		llama.AutoStartDriver = &auto
		changed = true
	}

	if !changed {
		return nil
	}
	existing.Drivers.Llama = llama
	out, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}
	header := "# Managed by Taiji — llama driver defaults for Ollama models.\n"
	if err := os.WriteFile(configPath, []byte(header+string(out)), 0o644); err != nil {
		return err
	}
	return nil
}

func discoverLlamaModelsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	patterns := []string{
		filepath.Join(home, "Dev", "Coditary", "plugins", "wuji", "llama", "models"),
		filepath.Join(home, "Dev", "*", "plugins", "wuji", "llama", "models"),
		filepath.Join(home, "Dev", "*", "*", "plugins", "wuji", "llama", "models"),
	}
	for _, pattern := range patterns {
		for _, dir := range globDirs(pattern) {
			if hasOllamaManifest(dir) {
				return dir
			}
		}
	}
	return ""
}

func discoverLlamaDriverBinary() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	name := "wuji-driver-llama"
	patterns := []string{
		filepath.Join(home, "Dev", "Coditary", "plugins", "wuji", "llama", "bin", name),
		filepath.Join(home, "Dev", "*", "plugins", "wuji", "llama", "bin", name),
		filepath.Join(home, "Dev", "*", "*", "plugins", "wuji", "llama", "bin", name),
	}
	for _, pattern := range patterns {
		for _, candidate := range globFiles(pattern) {
			if ok, _ := executableFile(candidate); ok {
				return candidate
			}
		}
	}
	if bin, ok := findExecutableInDirs(name, executableSearchDirs()); ok {
		return bin
	}
	candidate := filepath.Join(home, "bin", name)
	if ok, _ := executableFile(candidate); ok {
		return candidate
	}
	return ""
}

func hasOllamaManifest(modelsDir string) bool {
	_, err := os.Stat(filepath.Join(modelsDir, "ollama", "manifest.json"))
	return err == nil
}

func globFiles(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return matches
}

func globDirs(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	var out []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || !info.IsDir() {
			continue
		}
		out = append(out, match)
	}
	return out
}

func wujiRootEnv(root string) string {
	return fmt.Sprintf("WUJI_ROOT=%s", root)
}
