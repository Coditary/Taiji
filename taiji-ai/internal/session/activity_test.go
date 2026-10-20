package session

import (
	"testing"
	"time"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

func TestGroupEventsByTurn(t *testing.T) {
	all := []events.Event{
		{Type: events.TypeStep, Step: 1},
		{Type: events.TypeToolCall, Step: 1, ToolName: "shell"},
		{Type: events.TypeObservation, Step: 1, ToolName: "shell"},
		{Type: events.TypeDone, Step: 2},
		{Type: events.TypeToolCall, Step: 900, ToolName: "memory_revise"},
		{Type: events.TypeStep, Step: 1},
		{Type: events.TypeToolCall, Step: 1, ToolName: "decide"},
		{Type: events.TypeDone, Step: 1},
	}

	groups := groupEventsByTurn(all)
	if len(groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(groups))
	}
	if len(groups[0]) != 5 {
		t.Fatalf("first group = %d events, want 5 (includes post-done memory)", len(groups[0]))
	}
	if groups[0][1].ToolName != "shell" {
		t.Fatalf("first tool = %q", groups[0][1].ToolName)
	}
	if groups[1][1].ToolName != "decide" {
		t.Fatalf("second tool = %q", groups[1][1].ToolName)
	}
}

func TestTurnsWithActivity(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "run shell"},
		{Role: provider.RoleAssistant, Content: "done"},
		{Role: provider.RoleUser, Content: "decide"},
		{Role: provider.RoleAssistant, Content: "ok"},
	}
	all := []events.Event{
		{Type: events.TypeStep, Step: 1},
		{Type: events.TypeToolCall, Step: 1, ToolName: "shell", IsSkill: false},
		{Type: events.TypeObservation, Step: 1, ToolName: "shell", Text: "ok"},
		{Type: events.TypeDone, Step: 1},
		{Type: events.TypeStep, Step: 1},
		{Type: events.TypeToolCall, Step: 1, ToolName: "invoke_skill", IsSkill: true, SkillName: "buy"},
		{Type: events.TypeObservation, Step: 1, ToolName: "invoke_skill", IsSkill: true, SkillName: "buy"},
		{Type: events.TypeDone, Step: 1},
	}

	turns := TurnsWithActivity(messages, all)
	if len(turns) != 2 {
		t.Fatalf("turns = %d", len(turns))
	}
	if len(turns[0].Activity) != 2 {
		t.Fatalf("turn 0 activity = %d, want 2", len(turns[0].Activity))
	}
	if turns[0].Activity[0].ToolName != "shell" {
		t.Fatalf("turn 0 tool = %q", turns[0].Activity[0].ToolName)
	}
	if len(turns[1].Activity) != 2 || !turns[1].Activity[0].IsSkill {
		t.Fatalf("turn 1 activity = %#v", turns[1].Activity)
	}
}

func TestTurnsWithActivityUserSentAt(t *testing.T) {
	sent := time.Date(2026, 3, 20, 14, 30, 0, 0, time.UTC)
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "hi"},
		{Role: provider.RoleAssistant, Content: "hey"},
	}
	all := []events.Event{
		{Type: events.TypeStep, Step: 1, Timestamp: sent},
		{Type: events.TypeDone, Step: 1},
	}
	turns := TurnsWithActivity(messages, all)
	if len(turns) != 1 {
		t.Fatalf("turns = %d", len(turns))
	}
	if !turns[0].UserSentAt.Equal(sent) {
		t.Fatalf("user_sent_at = %v, want %v", turns[0].UserSentAt, sent)
	}
}

func TestMetricsFromEvents(t *testing.T) {
	m := metricsFromEvents([]events.Event{
		{Type: events.TypeDone, Step: 1},
		{Type: events.TypeTurnMetrics, Data: map[string]any{
			"total_duration_ms":    int64(2300),
			"thinking_duration_ms": int64(400),
			"tokens_per_sec":       42.5,
			"had_thinking":         true,
		}},
	})
	if m == nil || m.TotalDurationMs != 2300 || !m.HadThinking {
		t.Fatalf("metrics = %#v", m)
	}
}

func TestFilterActivityEventsIncludesMemoryRecall(t *testing.T) {
	filtered := filterActivityEvents([]events.Event{
		{Type: events.TypeStep, Step: 1},
		{Type: events.TypeMemoryRecalled, Step: 1, ToolName: "memory_recall", Text: "hit"},
		{Type: events.TypeToolCall, Step: 1, ToolName: "grep"},
	})
	if len(filtered) != 2 {
		t.Fatalf("filtered = %#v", filtered)
	}
	if filtered[0].ToolName != "memory_recall" || filtered[0].Type != events.TypeObservation {
		t.Fatalf("memory recall = %#v", filtered[0])
	}
}

func TestFilterActivityEventsDedupesMemoryRecall(t *testing.T) {
	filtered := filterActivityEvents([]events.Event{
		{Type: events.TypeToolCall, Step: 1, ToolName: "memory_recall"},
		{Type: events.TypeObservation, Step: 1, ToolName: "memory_recall", Text: "hit"},
		{Type: events.TypeMemoryRecalled, Step: 1, ToolName: "memory_recall", Text: "hit"},
	})
	if len(filtered) != 2 {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestFilterActivityEventsIncludesMemoryStored(t *testing.T) {
	filtered := filterActivityEvents([]events.Event{
		{Type: events.TypeMemoryStored, Step: 900, Text: "stored id=abc"},
	})
	if len(filtered) != 1 || filtered[0].ToolName != "memory_remember" {
		t.Fatalf("filtered = %#v", filtered)
	}
}
