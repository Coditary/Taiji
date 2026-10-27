package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

func TestHookEnvForAndAppendHookContext(t *testing.T) {
	s, err := session.New(t.TempDir(), "s1", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	env := hookEnvFor(RunRequest{Session: s, WorkDir: "/tmp/wd"}, 3, " goal ")
	if env.SessionID != "s1" || env.WorkDir != "/tmp/wd" || env.Step != 3 || env.Goal != "goal" {
		t.Fatalf("env=%+v", env)
	}
	msgs := []provider.Message{{Role: provider.RoleSystem, Content: "base"}}
	out := appendHookContext(msgs, "extra")
	if out[0].Content != "base\n\nextra" {
		t.Fatalf("content=%q", out[0].Content)
	}
	if got := appendHookContext(msgs, "  "); len(got) != 1 {
		t.Fatal("empty extra unchanged")
	}
	out = appendHookContext([]provider.Message{{Role: provider.RoleUser, Content: "hi"}}, "sys")
	if out[0].Role != provider.RoleSystem || out[0].Content != "sys" {
		t.Fatalf("prepend system: %+v", out[0])
	}
}

func TestRunSessionErrorHookNilSafe(t *testing.T) {
	runSessionErrorHook(context.Background(), RunRequest{}, hooks.Env{}, errors.New("boom"))
	runSessionErrorHook(context.Background(), RunRequest{Hooks: hooks.NewRunner(nil)}, hooks.Env{}, nil)
}
