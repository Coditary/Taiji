package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

func TestSearchConversationsToolFindsMessage(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { SetWorkspaceRootsForTest(".", "") })
	SetWorkspaceRootsForTest(root, "")
	sess, err := session.New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	sess.State.Messages = append(sess.State.Messages, provider.Message{
		Role: provider.RoleUser, Content: "unique-keyword-xyzzy plugh",
	})
	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	tool := SearchConversations{}
	out, err := tool.Run(context.Background(), map[string]any{
		"query": "xyzzy plugh",
		"limit": 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || out == "No matching conversations." {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, sess.State.ID) {
		t.Fatalf("missing session id in %q", out)
	}
}
