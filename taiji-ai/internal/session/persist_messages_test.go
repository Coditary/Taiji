package session

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestSanitizeStoredMessagesDropsFewShots(t *testing.T) {
	settings := Settings{
		FewShotExamples: []ExampleTurn{
			{Role: "user", Content: "Was passiert in der Szene?"},
			{Role: "assistant", Content: "Der Regen trommelte."},
		},
	}
	in := []provider.Message{
		{Role: provider.RoleSystem, Content: "system prompt"},
		{Role: provider.RoleUser, Content: "Was passiert in der Szene?"},
		{Role: provider.RoleAssistant, Content: "Der Regen trommelte."},
		{Role: provider.RoleUser, Content: "@story Ich gehe weiter"},
		{Role: provider.RoleSystem, Content: "[Turn focus]\nSpielleiter"},
		{Role: provider.RoleAssistant, Content: "Im Wald..."},
	}
	out := SanitizeStoredMessages(in, settings)
	if len(out) != 2 {
		t.Fatalf("got %d messages, want 2: %#v", len(out), out)
	}
	if out[0].Content != "@story Ich gehe weiter" || out[1].Content != "Im Wald..." {
		t.Fatalf("unexpected conversation: %#v", out)
	}
}

func TestApplyDisplayGoalNoOpWhenSame(t *testing.T) {
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "same"}}
	ApplyDisplayGoalToLastUser(msgs, "same", "same")
	if msgs[0].Content != "same" {
		t.Fatalf("content=%q", msgs[0].Content)
	}
}

func TestApplyDisplayGoalSuffixMatch(t *testing.T) {
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "prefix\n\ninner"}}
	ApplyDisplayGoalToLastUser(msgs, "@story inner", "inner")
	if msgs[0].Content != "@story inner" {
		t.Fatalf("content=%q", msgs[0].Content)
	}
}

func TestApplyDisplayGoalToLastUser(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "Ich gehe weiter"},
		{Role: provider.RoleAssistant, Content: "ok"},
	}
	ApplyDisplayGoalToLastUser(msgs, "@story Ich gehe weiter", "Ich gehe weiter")
	if msgs[0].Content != "@story Ich gehe weiter" {
		t.Fatalf("content = %q", msgs[0].Content)
	}
}
