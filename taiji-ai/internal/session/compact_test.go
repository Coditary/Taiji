package session

import (
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestCompactUserMessageRoundTrip(t *testing.T) {
	raw := FormatCompactUserMessage("did things\nand stuff")
	if !IsCompactUserMessage(raw) {
		t.Fatal("expected compact marker")
	}
	body, ok := ParseCompactUserMessage(raw)
	if !ok || body != "did things\nand stuff" {
		t.Fatalf("body = %q ok=%v", body, ok)
	}
}

func TestTurnsFromMessagesCompact(t *testing.T) {
	turns := TurnsFromMessages([]provider.Message{
		{Role: provider.RoleUser, Content: FormatCompactUserMessage("earlier")},
		{Role: provider.RoleUser, Content: "hello"},
		{Role: provider.RoleAssistant, Content: "hi"},
	})
	if len(turns) != 2 {
		t.Fatalf("turns = %d", len(turns))
	}
	if turns[0].UserKind != "compact" || turns[0].User != "earlier" {
		t.Fatalf("first = %#v", turns[0])
	}
	if turns[1].User != "hello" || turns[1].Assistant != "hi" {
		t.Fatalf("second = %#v", turns[1])
	}
}

func TestChatTurnsToMessagesCompact(t *testing.T) {
	msgs := chatTurnsToMessages([]ChatTurn{
		{User: "summary text", UserKind: "compact"},
		{User: "q", Assistant: "a"},
	})
	if len(msgs) != 3 {
		t.Fatalf("msgs = %d", len(msgs))
	}
	if !IsCompactUserMessage(msgs[0].Content) {
		t.Fatalf("first content = %q", msgs[0].Content)
	}
	if !strings.Contains(msgs[0].Content, "summary text") {
		t.Fatalf("missing summary: %q", msgs[0].Content)
	}
}
