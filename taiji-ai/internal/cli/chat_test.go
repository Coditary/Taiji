package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/session"
)

func TestResolveChatSessionResumesDirectoryBinding(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "src")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	app := &App{Config: &config.Config{Root: root}}
	first, resumed, err := resolveChatSession(app, chatOptions{}, workDir)
	if err != nil {
		t.Fatal(err)
	}
	if resumed {
		t.Fatal("expected new session on first use")
	}
	if err := first.Save(); err != nil {
		t.Fatal(err)
	}

	second, resumed, err := resolveChatSession(app, chatOptions{}, workDir)
	if err != nil {
		t.Fatal(err)
	}
	if !resumed {
		t.Fatal("expected resumed session for same directory")
	}
	if second.State.ID != first.State.ID {
		t.Fatalf("session ids differ: %q vs %q", second.State.ID, first.State.ID)
	}
}

func TestResolveChatSessionUsesSeparateBindingsPerDirectory(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	for _, dir := range []string{dirA, dirB} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	app := &App{Config: &config.Config{Root: root}}
	sessA, _, err := resolveChatSession(app, chatOptions{}, dirA)
	if err != nil {
		t.Fatal(err)
	}
	sessB, _, err := resolveChatSession(app, chatOptions{}, dirB)
	if err != nil {
		t.Fatal(err)
	}
	if sessA.State.ID == sessB.State.ID {
		t.Fatalf("expected different sessions for different directories")
	}
}

func TestHandleSlashCommandHelp(t *testing.T) {
	state := &chatState{
		app:     &App{Config: &config.Config{Root: t.TempDir()}},
		session: &session.Session{State: session.State{ID: "abc123"}},
	}
	handled, exit, err := state.handleSlashCommand("/help")
	if err != nil || !handled || exit {
		t.Fatalf("handled=%v exit=%v err=%v", handled, exit, err)
	}
}

func TestHandleSlashCommandClear(t *testing.T) {
	root := t.TempDir()
	state := &chatState{
		app:        &App{Config: &config.Config{Root: root}},
		session:    &session.Session{State: session.State{ID: "old"}},
		workDir:    root,
		providerID: "dummy",
	}
	oldID := state.session.State.ID
	handled, exit, err := state.handleSlashCommand("/clear")
	if err != nil || !handled || exit {
		t.Fatalf("handled=%v exit=%v err=%v", handled, exit, err)
	}
	if state.session.State.ID == oldID {
		t.Fatal("expected new session after /clear")
	}
}

func TestHandleSlashCommandExit(t *testing.T) {
	state := &chatState{
		app:     &App{Config: &config.Config{Root: t.TempDir()}},
		session: &session.Session{State: session.State{ID: "abc123"}},
	}

	for _, cmd := range []string{"/exit", "/quit"} {
		handled, exit, err := state.handleSlashCommand(cmd)
		if err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
		if !handled || !exit {
			t.Fatalf("%s: handled=%v exit=%v", cmd, handled, exit)
		}
	}
}

func TestHandleSlashCommandSession(t *testing.T) {
	state := &chatState{
		app:     &App{Config: &config.Config{Root: t.TempDir()}},
		session: &session.Session{State: session.State{ID: "abc123", Name: "Test chat"}},
	}

	handled, exit, err := state.handleSlashCommand("/session")
	if err != nil {
		t.Fatalf("handleSlashCommand: %v", err)
	}
	if !handled || exit {
		t.Fatalf("handled=%v exit=%v", handled, exit)
	}
}

func TestHandleSlashCommandName(t *testing.T) {
	root := t.TempDir()
	sess, err := session.New(root, "id1", "", "chat")
	if err != nil {
		t.Fatal(err)
	}
	state := &chatState{
		app:     &App{Config: &config.Config{Root: root}},
		session: sess,
	}

	handled, exit, err := state.handleSlashCommand("/name My Session")
	if err != nil {
		t.Fatal(err)
	}
	if !handled || exit {
		t.Fatalf("handled=%v exit=%v", handled, exit)
	}
	if sess.State.Name != "My Session" || !sess.State.NameManual {
		t.Fatalf("state = %+v", sess.State)
	}
}

func TestHandleSlashCommandDeleteCurrentSession(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "src")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sess, err := session.New(root, "old-session", "", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := session.SaveDirBinding(root, workDir, sess.State.ID); err != nil {
		t.Fatal(err)
	}

	state := &chatState{
		app:        &App{Config: &config.Config{Root: root}},
		providerID: "echo",
		workDir:    workDir,
		session:    sess,
	}

	handled, exit, err := state.handleSlashCommand("/delete")
	if err != nil {
		t.Fatal(err)
	}
	if !handled || exit {
		t.Fatalf("handled=%v exit=%v", handled, exit)
	}
	if state.session.State.ID == "old-session" {
		t.Fatal("expected new session after delete")
	}
	if _, err := os.Stat(sess.Dir); !os.IsNotExist(err) {
		t.Fatal("old session directory should be deleted")
	}
}

func TestHandleSlashCommandNotSlash(t *testing.T) {
	state := &chatState{
		app:     &App{Config: &config.Config{Root: t.TempDir()}},
		session: &session.Session{State: session.State{ID: "abc123"}},
	}

	handled, exit, err := state.handleSlashCommand("hello")
	if err != nil {
		t.Fatalf("handleSlashCommand: %v", err)
	}
	if handled || exit {
		t.Fatalf("handled=%v exit=%v", handled, exit)
	}
}
