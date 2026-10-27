package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestHandleSlashCommandToolsAndProvider(t *testing.T) {
	state := newDummyChatState(t)
	state.rt.Tools.Register(builtin.ListDir{})

	out := captureStderr(t, func() {
		handled, exit, err := state.handleSlashCommand("/tools")
		if err != nil || !handled || exit {
			t.Fatalf("tools: handled=%v exit=%v err=%v", handled, exit, err)
		}
	})
	if !strings.Contains(out, "list_dir") {
		t.Fatalf("missing list_dir in %q", out)
	}

	out = captureStderr(t, func() {
		handled, exit, err := state.handleSlashCommand("/provider")
		if err != nil || !handled || exit {
			t.Fatalf("provider show: %v", err)
		}
	})
	if !strings.Contains(out, "dummy_a") {
		t.Fatalf("provider output: %q", out)
	}

	if _, _, err := state.handleSlashCommand("/provider nope"); err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestHandleSlashCommandUnknownAndNameShow(t *testing.T) {
	state := newDummyChatState(t)
	out := captureStderr(t, func() {
		handled, exit, err := state.handleSlashCommand("/nope")
		if err != nil || !handled || exit {
			t.Fatalf("unknown: handled=%v exit=%v err=%v", handled, exit, err)
		}
	})
	if !strings.Contains(out, "unknown command") {
		t.Fatalf("out=%q", out)
	}

	out = captureStderr(t, func() {
		_, _, _ = state.handleSlashCommand("/name")
	})
	if !strings.Contains(out, "session name") {
		t.Fatalf("out=%q", out)
	}
}

func TestHandleSlashCommandSkillsNilCatalog(t *testing.T) {
	state := newDummyChatState(t)
	state.rt.Skills = nil
	out := captureStderr(t, func() {
		handled, exit, err := state.handleSlashCommand("/skills")
		if err != nil || !handled || exit {
			t.Fatalf("skills: %v", err)
		}
	})
	if !strings.Contains(out, "not configured") {
		t.Fatalf("out=%q", out)
	}
}

func TestHandleSlashCommandSkillsListsInvokable(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, ".taiji", "skills", "demo")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte(`---
name: demo-skill
description: Demo
---
Body`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy_a",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
		},
		Skills: taijicfg.SkillsConfig{Dirs: []string{filepath.Join(root, ".taiji", "skills")}},
	}
	reg, err := provider.NewRegistry(context.Background(), cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Quiet: true, Local: true, LocalOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	rt.Providers = reg

	state := &chatState{
		app:        &App{Config: cfg},
		rt:         rt,
		providerID: "dummy_a",
		workDir:    root,
		session:    &session.Session{State: session.State{ID: "s1"}},
	}
	out := captureStderr(t, func() {
		handled, exit, err := state.handleSlashCommand("/skills")
		if err != nil || !handled || exit {
			t.Fatalf("skills: %v", err)
		}
	})
	if !strings.Contains(out, "demo-skill") {
		t.Fatalf("out=%q", out)
	}
}

func TestResolveChatSessionExplicitID(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "wd")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sess, err := session.New(root, "explicit", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}
	app := &App{Config: &taijicfg.Config{Root: root}}
	got, resumed, err := resolveChatSession(app, chatOptions{sessionID: "explicit"}, workDir)
	if err != nil || !resumed || got.State.ID != "explicit" {
		t.Fatalf("sess=%v resumed=%v err=%v", got.State.ID, resumed, err)
	}
}

func TestMaybeAutoNameSessionDisabled(t *testing.T) {
	state := newDummyChatState(t)
	state.app.Config.Agent.AutoNameSessions = boolPtr(false)
	if err := maybeAutoNameSession(context.Background(), state, "hello"); err != nil {
		t.Fatal(err)
	}
}

func TestStdinIsTTY(t *testing.T) {
	// Pipe stdin is not a TTY in test harness.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old; _ = r.Close(); _ = w.Close() }()
	if stdinIsTTY() {
		t.Fatal("pipe should not be TTY")
	}
}

func TestPrintChatHelpToBuffer(t *testing.T) {
	var buf bytes.Buffer
	printChatHelp(&buf)
	if !strings.Contains(buf.String(), "/compact") {
		t.Fatal("help text")
	}
}

func boolPtr(b bool) *bool { return &b }
