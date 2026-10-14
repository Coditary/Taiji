package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeLegacyWujiPaths(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{
		Root: root,
		Wuji: WujiConfig{
			AutoSync: true,
			Endpoint: "unix:///home/user/Dev/Coditary-Bundle/wuji-ai/wuji-ai/.wuji/run/drivers/core.sock",
			WujiRoot: "../wuji-ai/wuji-ai",
		},
		Providers: map[string]ProviderConfig{
			"gemma4": {
				Type:       "wuji",
				Endpoint:   "unix:///home/user/Dev/Coditary-Bundle/wuji-ai/wuji-ai/.wuji/run/drivers/core.sock",
				WujiRoot:   "../wuji-ai/wuji-ai",
				ProviderID: "llama",
				Model:      "ollama/gemma4_26b-128k-nothink.gguf",
				Synced:     true,
			},
		},
	}
	sanitizeLegacyWujiPaths(cfg)
	if cfg.Wuji.Endpoint != "auto" || cfg.Wuji.WujiRoot != "" {
		t.Fatalf("wuji config = %+v", cfg.Wuji)
	}
	pc := cfg.Providers["gemma4"]
	if pc.Endpoint != "auto" || pc.WujiRoot != "" {
		t.Fatalf("provider config = %+v", pc)
	}
	if pc.Model != "ollama/gemma4_26b-128k-nothink.gguf" {
		t.Fatalf("model should be preserved: %q", pc.Model)
	}
}

func TestSanitizeKeepsValidHomeSocket(t *testing.T) {
	home := t.TempDir()
	sock := filepath.Join(home, ".wuji", "run", "drivers", "core.sock")
	if err := os.MkdirAll(filepath.Dir(sock), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		Root: home,
		Wuji: WujiConfig{
			Endpoint: "unix://" + sock,
		},
	}
	sanitizeLegacyWujiPaths(cfg)
	if cfg.Wuji.Endpoint != "unix://"+sock {
		t.Fatalf("endpoint changed: %q", cfg.Wuji.Endpoint)
	}
}
