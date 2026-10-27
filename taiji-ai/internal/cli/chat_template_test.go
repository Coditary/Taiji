package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/session"
)

func TestHandleSlashCommandTemplateList(t *testing.T) {
	root := t.TempDir()
	templates := filepath.Join(root, ".taiji", "templates")
	if err := os.MkdirAll(templates, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templates, "story.yaml"), []byte(`id: story
name: Story mode
`), 0o644); err != nil {
		t.Fatal(err)
	}
	state := &chatState{
		app:     &App{Config: &taijicfg.Config{Root: root}},
		session: &session.Session{State: session.State{ID: "s1"}},
	}
	out := captureStderr(t, func() {
		handled, exit, err := state.handleSlashCommand("/template")
		if err != nil || !handled || exit {
			t.Fatalf("handled=%v exit=%v err=%v", handled, exit, err)
		}
	})
	if !strings.Contains(out, "story") {
		t.Fatalf("out=%q", out)
	}
}
