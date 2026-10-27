package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/session"
)

func TestSessionsListCommand(t *testing.T) {
	root := t.TempDir()
	sess, err := session.New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	sess.State.Name = "Demo"
	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	app := &App{Config: &config.Config{Root: root}}
	cmd := newSessionsCmd(app)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	runErr := listCmd.RunE(listCmd, nil)
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	text := string(out)
	if !strings.Contains(text, sess.State.ID) || !strings.Contains(text, "Demo") {
		t.Fatalf("output=%q", text)
	}
}

func TestOrDash(t *testing.T) {
	if orDash("") != "-" || orDash("x") != "x" {
		t.Fatal("orDash mismatch")
	}
}
