package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

func TestSessionLoadSaveRoundTrip(t *testing.T) {
	root := t.TempDir()
	created, err := New(root, "persist-1", "openai", "build feature")
	if err != nil {
		t.Fatal(err)
	}
	created.State.Messages = []provider.Message{{Role: "user", Content: "hello"}}
	created.AddUsage(provider.Usage{
		InputTokens:      100,
		OutputTokens:     50,
		LatencyMs:        250,
		EstimatedCostUSD: 0.01,
	}, 2)
	if err := created.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(root, "persist-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State.Provider != "openai" || loaded.State.Goal != "build feature" {
		t.Fatalf("state = %+v", loaded.State)
	}
	if loaded.State.Stats.InputTokens != 100 || loaded.State.Stats.Steps != 2 {
		t.Fatalf("stats = %+v", loaded.State.Stats)
	}
	if len(loaded.State.Messages) != 1 {
		t.Fatalf("messages = %d", len(loaded.State.Messages))
	}
	statePath := filepath.Join(config.SessionsDir(root), "persist-1", "state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state.json: %v", err)
	}
}

func TestNewGeneratesIDWhenEmpty(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if s.State.ID == "" {
		t.Fatal("expected generated id")
	}
	if s.Dir != filepath.Join(config.SessionsDir(root), s.State.ID) {
		t.Fatalf("dir = %q", s.Dir)
	}
}

func TestLoadMissingSession(t *testing.T) {
	root := t.TempDir()
	if _, err := Load(root, "missing"); err == nil {
		t.Fatal("expected error for missing session")
	}
}

func TestAddUsageUpdatesStats(t *testing.T) {
	s := &Session{State: State{}}
	before := time.Now().UTC()
	s.AddUsage(provider.Usage{InputTokens: 3, OutputTokens: 7, LatencyMs: 11}, 1)
	if s.State.Stats.Requests != 1 || s.State.Stats.LastInputTokens != 3 {
		t.Fatalf("stats = %+v", s.State.Stats)
	}
	if s.State.Stats.UpdatedAt.Before(before) {
		t.Fatal("expected updated_at to be set")
	}
	s.AddUsage(provider.Usage{InputTokens: 0, OutputTokens: 1, LatencyMs: 2}, 5)
	if s.State.Stats.Steps != 5 {
		t.Fatalf("steps = %d", s.State.Stats.Steps)
	}
}
