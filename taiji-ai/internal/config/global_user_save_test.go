package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveGlobalUserPrefs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ui := UIConfig{Theme: "dark", Locale: "de"}
	stt := STTConfig{Model: "whisper-1"}
	if err := SaveGlobalUserPrefs(ui, stt); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(GlobalConfigDir(), configFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, "dark") || !strings.Contains(body, "whisper-1") {
		t.Fatalf("config=%q", body)
	}
}
