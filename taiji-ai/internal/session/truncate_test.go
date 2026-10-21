package session

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestTruncateAfterTurnDropsLaterTurnsAndAssistant(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "one"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "two"},
		{Role: provider.RoleAssistant, Content: "a2"},
		{Role: provider.RoleUser, Content: "three"},
		{Role: provider.RoleAssistant, Content: "a3"},
	}
	out := TruncateAfterTurn(messages, 1)
	turns := TurnsFromMessages(out)
	if len(turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(turns))
	}
	if turns[0].User != "one" || turns[0].Assistant != "a1" {
		t.Fatalf("first = %#v", turns[0])
	}
	if turns[1].User != "two" || turns[1].Assistant != "" {
		t.Fatalf("second = %#v", turns[1])
	}
}

func TestTruncateAfterTurnPreservesToolMessagesBeforeCut(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "one"},
		{Role: provider.RoleTool, Content: "tool-out", ToolCallID: "t1"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "two"},
		{Role: provider.RoleAssistant, Content: "a2"},
	}
	out := TruncateAfterTurn(messages, 0)
	if len(out) != 2 || out[0].Content != "one" || out[1].Role != provider.RoleTool {
		t.Fatalf("out = %#v", out)
	}
}

func TestTruncateAfterTurnNoOpWhenIndexBeyondEnd(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "one"},
		{Role: provider.RoleAssistant, Content: "a1"},
	}
	out := TruncateAfterTurn(messages, 5)
	if len(out) != len(messages) {
		t.Fatalf("len = %d", len(out))
	}
}

func TestVisibleTurnCount(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "one"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "two"},
	}
	if VisibleTurnCount(messages) != 2 {
		t.Fatalf("count = %d", VisibleTurnCount(messages))
	}
}
