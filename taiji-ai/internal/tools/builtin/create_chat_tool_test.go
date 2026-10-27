package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func TestCreateChatToolRegistersAndMetadata(t *testing.T) {
	tool := CreateChatTool{}
	if tool.Name() != "create_chat_tool" || tool.Description() == "" {
		t.Fatal("expected name and description")
	}
	if tool.Parameters()["type"] != "object" {
		t.Fatal("expected object parameters")
	}

	root := t.TempDir()
	sess, err := session.New(root, "chat-tool", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	ctx := session.WithActiveSession(context.Background(), sess)

	out, err := tool.Run(ctx, map[string]any{
		"name":        "summarize",
		"description": "Summarize text",
		"language":    "python",
		"script":      "print('ok')",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "summarize") {
		t.Fatalf("out=%q", out)
	}
	if len(sess.State.Settings.ChatTools) != 1 || sess.ChatToolNames()[0] != "summarize" {
		t.Fatalf("tools=%#v", sess.State.Settings.ChatTools)
	}
}

func TestEnableCreateChatToolRegisters(t *testing.T) {
	reg := tools.NewRegistry()
	EnableCreateChatTool(reg)
	if _, ok := reg.Get("create_chat_tool"); !ok {
		t.Fatal("expected create_chat_tool in registry")
	}
	EnableCreateChatTool(nil)
}

func TestCreateChatToolRequiresSession(t *testing.T) {
	_, err := (CreateChatTool{}).Run(context.Background(), map[string]any{
		"name": "x", "language": "shell", "script": "echo hi",
	})
	if err == nil {
		t.Fatal("expected error without session")
	}
}
