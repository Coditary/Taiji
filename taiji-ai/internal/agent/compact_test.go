package agent

import (
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

func TestHistoryCharCount(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "hello"},
		{Role: provider.RoleAssistant, Content: "world"},
	}
	if HistoryCharCount(msgs) != 10 {
		t.Fatalf("got %d", HistoryCharCount(msgs))
	}
}

func TestShouldAutoCompactHistory(t *testing.T) {
	off := 0
	cfg := &config.Config{Agent: config.AgentConfig{MaxContextChars: &off}}
	if ShouldAutoCompactHistory(nil, cfg) {
		t.Fatal("0 should disable")
	}
	limit := 10
	cfg.Agent.MaxContextChars = &limit
	msgs := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("x", 20)}}
	if !ShouldAutoCompactHistory(msgs, cfg) {
		t.Fatal("expected over limit")
	}
}

func TestDeterministicCompactSummary(t *testing.T) {
	turns := []session.ChatTurn{
		{User: "fix bug", Assistant: "done"},
	}
	out := deterministicCompactSummary(turns, "### todo_plan\n[ ] 1. ship")
	if !strings.Contains(out, "fix bug") || !strings.Contains(out, "todo_plan") {
		t.Fatalf("%s", out)
	}
}
