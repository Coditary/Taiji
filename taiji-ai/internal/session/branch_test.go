package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

func TestTruncateBeforeTurnDropsLaterTurnsKeepsAssistant(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "one"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "two"},
		{Role: provider.RoleAssistant, Content: "a2"},
	}
	out := TruncateBeforeTurn(messages, 1)
	turns := TurnsFromMessages(out)
	if len(turns) != 1 {
		t.Fatalf("turns = %d", len(turns))
	}
	if turns[0].Assistant != "a1" {
		t.Fatalf("turn = %#v", turns[0])
	}
}

func TestBranchFromCopiesThroughTurn(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src, err := New(root, "src", "p", "chat")
	if err != nil {
		t.Fatal(err)
	}
	src.State.Messages = []provider.Message{
		{Role: provider.RoleUser, Content: "one"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "two"},
		{Role: provider.RoleAssistant, Content: "a2"},
	}
	if err := src.Save(); err != nil {
		t.Fatal(err)
	}

	br, err := BranchFrom(root, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	turns := TurnsFromMessages(br.State.Messages)
	if len(turns) != 1 || turns[0].User != "one" {
		t.Fatalf("branch = %#v", turns)
	}
	if br.State.ID == src.State.ID {
		t.Fatal("expected new session id")
	}
}

func TestBranchFromCopiesEventsThroughTurn(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src, err := New(root, "src", "p", "chat")
	if err != nil {
		t.Fatal(err)
	}
	src.State.Messages = []provider.Message{
		{Role: provider.RoleUser, Content: "one"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "two"},
		{Role: provider.RoleAssistant, Content: "a2"},
	}
	ts := time.Now().UTC()
	sid := src.State.ID
	evts := []events.Event{
		{Type: events.TypeStep, SessionID: sid, Timestamp: ts},
		{Type: events.TypeTurnMetrics, SessionID: sid, Timestamp: ts, Data: map[string]any{"total_duration_ms": 100}},
		{Type: events.TypeDone, SessionID: sid, Timestamp: ts},
		{Type: events.TypeStep, SessionID: sid, Timestamp: ts},
		{Type: events.TypeDone, SessionID: sid, Timestamp: ts},
	}
	if err := events.WriteFile(filepath.Join(src.Dir, "events.jsonl"), evts); err != nil {
		t.Fatal(err)
	}

	br, err := BranchFrom(root, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := events.ReadFile(filepath.Join(br.Dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(copied) != 3 {
		t.Fatalf("events = %d, want 3", len(copied))
	}
	for _, ev := range copied {
		if ev.SessionID != br.State.ID {
			t.Fatalf("session_id = %q want %q", ev.SessionID, br.State.ID)
		}
	}
	turns := TurnsWithActivity(br.State.Messages, copied)
	if len(turns) != 1 || turns[0].Metrics == nil {
		t.Fatalf("turns = %#v", turns)
	}
}
