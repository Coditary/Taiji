package agent

import (
	"context"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestRunMemoryReflectionAfterToolError(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = root + "/global-memory"
	svc := memory.NewService(cfg, nil)

	reg := tools.NewRegistry()
	builtin.EnableMemoryOrchestration(reg, svc)
	reg.Register(errorEchoTool{name: "grep"})

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: `[tool call] grep {"pattern": "missing"}`},
			{Content: "I noted the failure."},
			{ToolCalls: []provider.ToolCall{{
				ID: "mem1", Name: "memory_remember",
				Args: `{"kind":"error","title":"grep miss","recall_text":"pattern missing","verified_fix":"widen pattern"}`,
			}}},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	s, err := session.New(root, "mem-reflect", "seq", "chat")
	if err != nil {
		t.Fatal(err)
	}
	memOn := true
	s.State.Settings.MemoryEnabled = &memOn

	_, err = Run(context.Background(), RunRequest{
		Goal:      "grep for missing pattern",
		Providers: providers,
		Tools:     reg,
		Config:    cfg,
		Memory:    svc,
		Session:   s,
		WorkDir:   root,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq.calls < 3 {
		t.Fatalf("expected reflection provider call, got %d", seq.calls)
	}
}

func TestRunMemoryReflectionPassDirect(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = root + "/global-memory"
	svc := memory.NewService(cfg, nil)
	reg := tools.NewRegistry()
	builtin.EnableMemoryOrchestration(reg, svc)

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{ToolCalls: []provider.ToolCall{{
				ID: "m1", Name: "memory_remember",
				Args: `{"kind":"pattern","title":"t","recall_text":"r","verified_fix":"v"}`,
			}}},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	tracker := newMemoryRunTracker()
	tracker.recordToolCall("grep", "{}", true, false)

	toolsDefs := filterProviderTools(reg, session.AgentRuntime{})
	msgs, err := runMemoryReflectionPass(
		context.Background(), seq, RunRequest{
			Goal: "g", Providers: providers, Tools: reg, Config: cfg, Memory: svc,
		},
		[]provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		toolsDefs,
		tracker,
		func(events.Event) {},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !tracker.storedMemory() {
		t.Fatal("expected memory stored via reflection pass")
	}
	if len(msgs) <= 1 {
		t.Fatalf("messages=%d", len(msgs))
	}
}
