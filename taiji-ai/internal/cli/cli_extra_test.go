package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/session"
)

func TestProvidersListCommand(t *testing.T) {
	cfg := &taijicfg.Config{
		Root:            t.TempDir(),
		DefaultProvider: "dummy_a",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
		},
	}
	app := &App{Config: cfg}
	cmd := newProvidersCmd(app)
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
	if !strings.Contains(string(out), "dummy_a") {
		t.Fatalf("out=%q", out)
	}
}

func TestSessionsDeleteCommand(t *testing.T) {
	root := t.TempDir()
	sess, err := session.New(root, "gone", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}
	app := &App{Config: &taijicfg.Config{Root: root}}
	cmd := newSessionsCmd(app)
	delCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	runErr := delCmd.RunE(delCmd, []string{sess.State.ID})
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !strings.Contains(string(out), "deleted") {
		t.Fatalf("out=%q", out)
	}
}

func TestSessionsListEmpty(t *testing.T) {
	app := &App{Config: &taijicfg.Config{Root: t.TempDir()}}
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
	if !strings.Contains(string(out), "no sessions") {
		t.Fatalf("out=%q", out)
	}
}

func TestProvidersUseMissingProvider(t *testing.T) {
	app := &App{Config: &taijicfg.Config{Root: t.TempDir(), Providers: map[string]taijicfg.ProviderConfig{}}}
	cmd := newProvidersCmd(app)
	useCmd, _, err := cmd.Find([]string{"use"})
	if err != nil {
		t.Fatal(err)
	}
	if err := useCmd.RunE(useCmd, []string{"missing"}); err == nil {
		t.Fatal("expected not found error")
	}
}
