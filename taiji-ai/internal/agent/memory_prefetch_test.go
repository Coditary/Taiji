package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/runtime"
)

func TestMemoryPrefetchStorePersona(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	var obs []string
	goal, result := runMemoryPrefetch(context.Background(), RunRequest{
		Memory: rt.Memory,
	}, "Du heißt jetzt Luma", func(ev events.Event) {
		if ev.Text != "" {
			obs = append(obs, ev.Text)
		}
	})

	if !result.stored {
		t.Fatal("expected auto store")
	}
	if !strings.Contains(goal, "[memory] Stored") {
		t.Fatalf("goal missing store preface: %q", goal)
	}

	entries, err := rt.Memory.Recall(context.Background(), memory.RecallRequest{
		Query: "assistant name persona",
		Kind:  string(memory.KindPersona),
		TopK:  3,
	})
	if err != nil || len(entries) == 0 {
		t.Fatalf("recall after prefetch: err=%v entries=%d", err, len(entries))
	}
	if !strings.Contains(strings.ToLower(entries[0].VerifiedFix), "luma") {
		t.Fatalf("expected Luma in recall, got %q", entries[0].VerifiedFix)
	}
}

func TestMemoryPrefetchRecallEmitsToolEvents(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	var toolCalls, observations int
	_, _ = runMemoryPrefetch(context.Background(), RunRequest{Memory: rt.Memory}, "Wie heißt du?", func(ev events.Event) {
		switch ev.Type {
		case events.TypeToolCall:
			if ev.ToolName == "memory_recall" {
				toolCalls++
			}
		case events.TypeObservation:
			if ev.ToolName == "memory_recall" {
				observations++
			}
		}
	})
	if toolCalls != 1 || observations != 1 {
		t.Fatalf("expected recall tool_call+observation, got calls=%d obs=%d", toolCalls, observations)
	}
}

func TestMemoryPrefetchRecallUserName(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	ctx := context.Background()
	_, err = rt.Memory.Remember(ctx, memory.RememberRequest{
		Kind:        string(memory.KindPreference),
		Title:       "User name: Leonard",
		RecallText:  "The user's name is Leonard.",
		VerifiedFix: "Leonard",
		Scope:       "local",
	})
	if err != nil {
		t.Fatal(err)
	}

	goal, result := runMemoryPrefetch(ctx, RunRequest{Memory: rt.Memory}, "Wie heiße ich?", func(events.Event) {})
	if !result.recalled {
		t.Fatal("expected auto recall")
	}
	if !strings.Contains(goal, "Leonard") {
		t.Fatalf("prefetch goal missing recalled name: %q", goal)
	}
}
