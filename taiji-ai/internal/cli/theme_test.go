package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestThemeInitWritesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	root := newThemeCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"init", "my-theme", "--force"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	path := filepath.Join(taijicfg.GlobalUIThemesDir(), "my-theme.theme")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "@theme {") {
		t.Fatal("expected @theme block in template")
	}
	if !strings.Contains(text, "my-theme") {
		t.Fatal("expected theme name in header comment")
	}
}
