package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/config"
)

func TestImportCardCommandWritesTemplate(t *testing.T) {
	root := t.TempDir()
	cardPath := filepath.Join(root, "hero.json")
	raw := `{"data":{"name":"Hero","description":"Brave","first_mes":"Hi"}}`
	if err := os.WriteFile(cardPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	app := &App{Config: &config.Config{Root: root}}
	cmd := newImportCardCmd(app)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := cmd.RunE(cmd, []string{cardPath})
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "Wrote template:") || !strings.Contains(text, "hero") {
		t.Fatalf("stdout=%q", text)
	}
}

func TestImportCardRequiresConfig(t *testing.T) {
	cmd := newImportCardCmd(&App{})
	if err := cmd.RunE(cmd, []string{"x.json"}); err == nil {
		t.Fatal("expected config error")
	}
}
