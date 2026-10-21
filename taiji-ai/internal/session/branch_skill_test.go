package session

import (
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
)

func TestBranchThroughFirstTurnKeepsSkillAnswer(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src, err := New(root, "src", "p", "chat")
	if err != nil {
		t.Fatal(err)
	}
	src.State.Messages = []provider.Message{
		{Role: provider.RoleUser, Content: "Hallo, sag mal hallo zu Tim"},
		{Role: provider.RoleAssistant, Content: "", ToolCalls: []provider.ToolCall{{ID: "c1", Name: skills.InvokeSkillToolName, Args: "{}"}}},
		{Role: provider.RoleTool, Name: skills.InvokeSkillToolName, Content: "## Skill: hallo\n\nBody"},
		{Role: provider.RoleSystem, Content: skills.ContinuationPrompt},
		{Role: provider.RoleUser, Content: "Hallo, sag mal hallo zu Tim"},
		{Role: provider.RoleAssistant, Content: "Hallo Tim! 👋"},
	}

	br, err := BranchFrom(root, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	turns := TurnsFromMessages(br.State.Messages)
	if len(turns) != 1 || turns[0].Assistant != "Hallo Tim! 👋" {
		t.Fatalf("branch messages = %#v", turns)
	}
}

func TestEventsBeforeTurnDropsRegeneratedTail(t *testing.T) {
	evts := []events.Event{
		{Type: events.TypeStep},
		{Type: events.TypeDone, Text: "first"},
		{Type: events.TypeStep},
		{Type: events.TypeDone, Text: "second"},
	}
	kept := EventsBeforeTurn(evts, 1)
	if len(kept) != 2 {
		t.Fatalf("kept = %d", len(kept))
	}
	if kept[1].Text != "first" {
		t.Fatalf("kept = %#v", kept)
	}
}
