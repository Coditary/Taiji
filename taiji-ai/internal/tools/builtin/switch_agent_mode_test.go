package builtin

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/session"
)

func TestSwitchAgentModeUpdatesSession(t *testing.T) {
	root := t.TempDir()
	sess, err := session.New(root, "mode", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	ctx := session.WithActiveSession(context.Background(), sess)

	out, err := (SwitchAgentMode{}).Run(ctx, map[string]any{"mode": "plan"})
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || sess.State.Settings.InteractionMode != "plan" {
		t.Fatalf("mode=%q out=%q", sess.State.Settings.InteractionMode, out)
	}

	out, err = (SwitchAgentMode{}).Run(ctx, map[string]any{"mode": "AGENT"})
	if err != nil {
		t.Fatal(err)
	}
	if sess.State.Settings.InteractionMode != "agent" {
		t.Fatalf("mode=%q out=%q", sess.State.Settings.InteractionMode, out)
	}
}

func TestSwitchAgentModeRequiresSession(t *testing.T) {
	_, err := (SwitchAgentMode{}).Run(context.Background(), map[string]any{"mode": "agent"})
	if err == nil {
		t.Fatal("expected error without session")
	}
}

func TestSwitchAgentModeRejectsUnknown(t *testing.T) {
	root := t.TempDir()
	sess, err := session.New(root, "bad-mode", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	ctx := session.WithActiveSession(context.Background(), sess)
	_, err = (SwitchAgentMode{}).Run(ctx, map[string]any{"mode": "debug"})
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
}
