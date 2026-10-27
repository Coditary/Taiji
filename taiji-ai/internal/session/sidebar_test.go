package session

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
)

func TestIsDraft(t *testing.T) {
	if IsDraft(nil) {
		t.Fatal("nil session is not a draft")
	}
	if !IsDraft(&Session{State: State{Messages: nil}}) {
		t.Fatal("empty messages should be draft")
	}
	if IsDraft(&Session{State: State{Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}}}}) {
		t.Fatal("session with messages is not a draft")
	}
	if !IsDraft(&Session{State: State{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "internal"},
	}}}) {
		t.Fatal("system-only session should still be a draft")
	}
}

func TestTurnsFromMessages(t *testing.T) {
	turns := TurnsFromMessages([]provider.Message{
		{Role: provider.RoleUser, Content: "hello"},
		{Role: provider.RoleAssistant, Content: "hi"},
		{Role: provider.RoleUser, Content: "again"},
		{Role: provider.RoleTool, Content: "ignored"},
		{Role: provider.RoleAssistant, Content: "sure"},
	})
	if len(turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(turns))
	}
	if turns[0].User != "hello" || turns[0].Assistant != "hi" {
		t.Fatalf("first turn = %#v", turns[0])
	}
	if turns[1].User != "again" || turns[1].Assistant != "sure" {
		t.Fatalf("second turn = %#v", turns[1])
	}
}

func TestTurnsFromMessagesSkipsLegacySkillContinuationUser(t *testing.T) {
	skillBody := "Skill instructions are loaded below. Use them internally.\n\n## Skill: hallo\n\nAntworte kurz."
	turns := TurnsFromMessages([]provider.Message{
		{Role: provider.RoleUser, Content: "Hallo"},
		{Role: provider.RoleAssistant, Content: ""},
		{Role: provider.RoleTool, Name: skills.InvokeSkillToolName, Content: "## Skill: hallo\n\nBody"},
		{Role: provider.RoleUser, Content: skillBody},
		{Role: provider.RoleAssistant, Content: "Hallo! Wie kann ich dir heute helfen?"},
	})
	if len(turns) != 1 {
		t.Fatalf("turns = %d, want 1: %#v", len(turns), turns)
	}
	if turns[0].User != "Hallo" || turns[0].Assistant != "Hallo! Wie kann ich dir heute helfen?" {
		t.Fatalf("turn = %#v", turns[0])
	}
}

func TestTurnsFromMessagesSkipsHiddenInternalUserPrompts(t *testing.T) {
	turns := TurnsFromMessages([]provider.Message{
		{Role: provider.RoleUser, Content: "Entscheide bitte mit dem Tool"},
		{Role: provider.RoleAssistant, Content: "Ich rufe decide auf."},
		{Role: provider.RoleSystem, Content: "## Memory reflection (mandatory)"},
		{Role: provider.RoleUser, Content: MemoryReflectionUserPrompt},
		{Role: provider.RoleAssistant, Content: "", ToolCalls: []provider.ToolCall{{ID: "call_1", Name: "memory_remember", Args: "{}"}}},
		{Role: provider.RoleTool, Name: "memory_remember", Content: "remembered id=test"},
	})
	if len(turns) != 1 {
		t.Fatalf("turns = %d, want 1: %#v", len(turns), turns)
	}
	if turns[0].User != "Entscheide bitte mit dem Tool" || turns[0].Assistant != "Ich rufe decide auf." {
		t.Fatalf("turn = %#v", turns[0])
	}
}

func TestTurnsFromMessagesMergesRepeatUserPromptAfterSkill(t *testing.T) {
	turns := TurnsFromMessages([]provider.Message{
		{Role: provider.RoleUser, Content: "Hallo, sag mal hallo zu Tim"},
		{Role: provider.RoleAssistant, Content: "", ToolCalls: []provider.ToolCall{{ID: "c1", Name: skills.InvokeSkillToolName, Args: "{}"}}},
		{Role: provider.RoleTool, Name: skills.InvokeSkillToolName, Content: "## Skill: hallo\n\nBody"},
		{Role: provider.RoleSystem, Content: skills.ContinuationPrompt},
		{Role: provider.RoleUser, Content: "Hallo, sag mal hallo zu Tim"},
		{Role: provider.RoleAssistant, Content: "Hallo Tim! 👋"},
	})
	if len(turns) != 1 {
		t.Fatalf("turns = %d, want 1: %#v", len(turns), turns)
	}
	if turns[0].Assistant != "Hallo Tim! 👋" {
		t.Fatalf("turn = %#v", turns[0])
	}
}

func TestTurnsFromMessagesIgnoresSkillContinuationSystem(t *testing.T) {
	turns := TurnsFromMessages([]provider.Message{
		{Role: provider.RoleUser, Content: "Hallo"},
		{Role: provider.RoleAssistant, Content: ""},
		{Role: provider.RoleTool, Name: skills.InvokeSkillToolName, Content: "## Skill: hallo\n\nBody"},
		{Role: provider.RoleSystem, Content: skills.ContinuationPrompt},
		{Role: provider.RoleAssistant, Content: "Hallo! Wie kann ich dir heute helfen?"},
	})
	if len(turns) != 1 {
		t.Fatalf("turns = %d, want 1: %#v", len(turns), turns)
	}
	if turns[0].User != "Hallo" || turns[0].Assistant != "Hallo! Wie kann ich dir heute helfen?" {
		t.Fatalf("turn = %#v", turns[0])
	}
}
