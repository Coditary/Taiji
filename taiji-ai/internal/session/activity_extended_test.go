package session

import (
	"testing"
	"time"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

func TestTurnsWithActivityFillsAssistantAndChrome(t *testing.T) {
	ts := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "hi"},
		{Role: provider.RoleAssistant, Content: ""},
	}
	evts := []events.Event{
		{Type: events.TypeStep, Step: 1, Timestamp: ts},
		{Type: events.TypeActorSwitch, Step: 1, Text: "story", Data: map[string]any{"template_display_name": "GM"}},
		{Type: events.TypeDone, Step: 1, Text: "Hello there"},
		{Type: events.TypeTurnMetrics, Step: 1, Data: map[string]any{"total_duration_ms": int64(42)}},
	}
	turns := TurnsWithActivity(msgs, evts)
	if len(turns) != 1 {
		t.Fatalf("turns=%d", len(turns))
	}
	if turns[0].Assistant != "Hello there" {
		t.Fatalf("assistant=%q", turns[0].Assistant)
	}
	if turns[0].AssistantAuthorLabel != "GM" {
		t.Fatalf("author=%q", turns[0].AssistantAuthorLabel)
	}
	if turns[0].UserSentAt != ts {
		t.Fatalf("sentAt=%v", turns[0].UserSentAt)
	}
	if turns[0].Metrics == nil || turns[0].Metrics.TotalDurationMs != 42 {
		t.Fatalf("metrics=%#v", turns[0].Metrics)
	}
}

func TestMergeActorState(t *testing.T) {
	if MergeActorState(nil, nil) != nil {
		t.Fatal("both nil")
	}
	got := MergeActorState(map[string]string{"a": "1"}, map[string]string{"b": "2", "a": "3"})
	if got["a"] != "3" || got["b"] != "2" {
		t.Fatalf("merge=%v", got)
	}
}
