package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestRunTurnRouterCodePrefetchAndMemory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "internal", "rag", "service.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package rag\n// RAG service\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = root + "/global-memory"
	svc := memory.NewService(cfg, nil)

	reg := tools.NewRegistry()
	reg.Register(builtin.Grep{})

	seq := &sequenceProvider{
		id:        "seq",
		responses: []provider.CompletionResponse{{Content: "The RAG code lives in internal/rag."}},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	s, err := session.New(root, "router-run", "seq", "chat")
	if err != nil {
		t.Fatal(err)
	}
	memOn := true
	s.State.Settings.MemoryEnabled = &memOn

	var routeSeen bool
	_, err = Run(context.Background(), RunRequest{
		Goal:      "Where is the RAG code in this repo?",
		Providers: providers,
		Tools:     reg,
		Config:    cfg,
		Memory:    svc,
		Session:   s,
		WorkDir:   root,
		EventHandler: func(ev events.Event) {
			if ev.ToolName == turnRouteToolName {
				routeSeen = true
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !routeSeen {
		t.Fatal("expected turn_route event")
	}
	if !strings.Contains(s.State.Messages[len(s.State.Messages)-1].Content, "RAG") {
		t.Fatal("expected answer about RAG")
	}
}

func TestRunDeterministicMemoryRevisionAfterSuccessfulRecall(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = root + "/global-memory"
	svc := memory.NewService(cfg, nil)
	ctx := context.Background()
	entry, err := svc.Remember(ctx, memory.RememberRequest{
		Kind:        string(memory.KindError),
		Title:       "Tool failure",
		RecallText:  "simulated failure",
		VerifiedFix: "retry with wider scope",
		Fingerprint: "sim:fail",
		Scope:       "local",
	})
	if err != nil {
		t.Fatal(err)
	}

	reg := tools.NewRegistry()
	reg.Register(errorEchoTool{name: "grep"})

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: `[tool call] grep {"pattern": "x"}`},
			{Content: "Fixed using the memory hint."},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	s, err := session.New(root, "mem-revise", "seq", "chat")
	if err != nil {
		t.Fatal(err)
	}
	memOn := true
	s.State.Settings.MemoryEnabled = &memOn

	var revised bool
	_, err = Run(context.Background(), RunRequest{
		Goal:      "grep for x",
		Providers: providers,
		Tools:     reg,
		Config:    cfg,
		Memory:    svc,
		Session:   s,
		WorkDir:   root,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
		EventHandler: func(ev events.Event) {
			if ev.Type == events.TypeObservation && ev.ToolName == "memory_revise" && strings.Contains(ev.Text, entry.ID) {
				revised = true
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !revised {
		t.Fatal("expected deterministic memory_revise observation")
	}
}
