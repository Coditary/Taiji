package dynamic

import (
	"context"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func TestSessionChatToolShellEcho(t *testing.T) {
	tool := SessionChatTool{
		WorkDir: t.TempDir(),
		Def: session.ChatTool{
			Name:        "echo_tool",
			Description: "echo",
			Language:    "shell",
			Script:      "#!/bin/sh\necho \"pong\"",
		},
	}
	out, err := tool.Run(context.Background(), map[string]any{"input": "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "pong" {
		t.Fatalf("out=%q", out)
	}
}

func TestSessionChatToolUnsupportedLanguage(t *testing.T) {
	tool := SessionChatTool{
		WorkDir: t.TempDir(),
		Def: session.ChatTool{
			Name:     "bad",
			Language: "ruby",
			Script:   "puts 1",
		},
	}
	_, err := tool.Run(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("err=%v", err)
	}
}

func TestExtendRegistryAddsSessionTools(t *testing.T) {
	base := tools.NewRegistry()
	sess := &session.Session{}
	sess.State.Settings.ChatTools = []session.ChatTool{
		{Name: "mytool", Language: "shell", Script: "true"},
	}
	extended := ExtendRegistry(base, sess, t.TempDir())
	names := extended.List()
	if len(names) != 1 || names[0].Name() != "mytool" {
		t.Fatalf("tools=%v", names)
	}
}

func TestExtendRegistryNilSessionReturnsBase(t *testing.T) {
	base := tools.NewRegistry()
	if ExtendRegistry(base, nil, "") != base {
		t.Fatal("expected same registry")
	}
}
