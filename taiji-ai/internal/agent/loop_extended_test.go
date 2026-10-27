package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

type errorEchoTool struct {
	name string
}

func (e errorEchoTool) Name() string               { return e.name }
func (e errorEchoTool) Description() string        { return "always errors" }
func (e errorEchoTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (e errorEchoTool) Run(context.Context, map[string]any) (string, error) {
	return "", fmt.Errorf("simulated failure")
}

func TestRunTurnRouterStopShortCircuits(t *testing.T) {
	var types []events.Type
	result, err := Run(context.Background(), RunRequest{
		Goal:      "stop",
		Providers: testDummyRegistry(t),
		Tools:     tools.NewRegistry(),
		Config:    &taijicfg.Config{},
		EventHandler: func(ev events.Event) {
			types = append(types, ev.Type)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps != 1 || result.Answer == "" {
		t.Fatalf("result=%+v", result)
	}
	hasDone := false
	for _, ty := range types {
		if ty == events.TypeDone {
			hasDone = true
		}
	}
	if !hasDone {
		t.Fatalf("events=%v", types)
	}
}

func TestRunPromptBeforeHookMutatesGoal(t *testing.T) {
	hooks.RegisterPlugin("agentPromptMutate", func(ctx context.Context, input hooks.ShellInput) (hooks.Result, error) {
		if input.Prompt != nil {
			input.Prompt.Text = "mutated goal"
		}
		return hooks.Result{}, nil
	})
	runner := hooks.NewRunner([]hooks.Entry{{
		ID: "pm", Event: hooks.EventPromptBefore, Actions: []hooks.ActionSpec{{Plugin: "agentPromptMutate"}},
	}})

	var capturedUser string
	wrapped := &capturingProvider{
		id: "cap",
		complete: func(_ context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
			for _, m := range req.Messages {
				if m.Role == provider.RoleUser {
					capturedUser = m.Content
				}
			}
			return &provider.CompletionResponse{Content: "ok"}, nil
		},
	}
	providers := provider.NewStaticRegistry("cap", map[string]provider.Provider{"cap": wrapped})

	_, err := Run(context.Background(), RunRequest{
		Goal:      "original",
		Providers: providers,
		Tools:     tools.NewRegistry(),
		Config:    &taijicfg.Config{},
		Hooks:     runner,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capturedUser, "mutated goal") {
		t.Fatalf("user content=%q", capturedUser)
	}
}

func TestRunSessionIdleHookFires(t *testing.T) {
	var idleAnswer string
	hooks.RegisterPlugin("agentSessionIdle", func(ctx context.Context, input hooks.ShellInput) (hooks.Result, error) {
		if input.Session != nil {
			idleAnswer = input.Session.Answer
		}
		return hooks.Result{}, nil
	})
	runner := hooks.NewRunner([]hooks.Entry{{
		ID: "idle", Event: hooks.EventSessionIdle, Actions: []hooks.ActionSpec{{Plugin: "agentSessionIdle"}},
	}})

	seq := &sequenceProvider{
		id:        "seq",
		responses: []provider.CompletionResponse{{Content: "final idle"}},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	_, err := Run(context.Background(), RunRequest{
		Goal:      "work",
		Providers: providers,
		Tools:     tools.NewRegistry(),
		Config:    &taijicfg.Config{},
		Hooks:     runner,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if idleAnswer != "final idle" {
		t.Fatalf("idleAnswer=%q", idleAnswer)
	}
}

type failTestsTool struct{}

func (failTestsTool) Name() string               { return "run_tests" }
func (failTestsTool) Description() string        { return "fake failing tests" }
func (failTestsTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (failTestsTool) Run(context.Context, map[string]any) (string, error) {
	return "❌ FAIL: 1 failed\nFAIL: TestFoo (pkg/a)", nil
}

func TestRunThrashingAppendsCircuitBreaker(t *testing.T) {
	dir := t.TempDir()
	reg := tools.NewRegistry()
	reg.Register(failTestsTool{})

	responses := make([]provider.CompletionResponse, 0, 4)
	for i := 0; i < 4; i++ {
		responses = append(responses, provider.CompletionResponse{
			Content: `[tool call] run_tests {}`,
		})
	}
	responses = append(responses, provider.CompletionResponse{Content: "gave up"})

	seq := &sequenceProvider{id: "seq", responses: responses}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	result, err := Run(context.Background(), RunRequest{
		Goal:      "search repeatedly",
		Providers: providers,
		Tools:     reg,
		Config:    &taijicfg.Config{Agent: taijicfg.AgentConfig{ThrashingThreshold: 3}},
		WorkDir:   dir,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	foundBreaker := false
	for _, m := range result.Messages {
		if m.Role == provider.RoleTool && strings.Contains(m.Content, "RUNTIME CIRCUIT BREAKER") {
			foundBreaker = true
		}
	}
	if !foundBreaker {
		t.Fatal("expected thrashing circuit breaker in tool observation")
	}
}

func TestRunMemoryAutoRecallOnToolError(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = root + "/global-memory"
	svc := memory.NewService(cfg, nil)
	ctx := context.Background()
	if _, err := svc.Remember(ctx, memory.RememberRequest{
		Kind:        string(memory.KindError),
		Title:       "grep failed",
		RecallText:  "pattern not found in repo",
		VerifiedFix: "Try a broader pattern",
		Fingerprint: "grep:notfound",
		Scope:       "local",
	}); err != nil {
		t.Fatal(err)
	}

	reg := tools.NewRegistry()
	reg.Register(errorEchoTool{name: "grep"})

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: `[tool call] grep {"pattern": "missing"}`},
			{Content: "handled"},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	s, err := session.New(root, "mem-recall", "seq", "chat")
	if err != nil {
		t.Fatal(err)
	}
	memOn := true
	s.State.Settings.MemoryEnabled = &memOn

	var recalled bool
	_, err = Run(context.Background(), RunRequest{
		Goal:      "grep for missing",
		Providers: providers,
		Tools:     reg,
		Config:    cfg,
		Memory:    svc,
		Session:   s,
		WorkDir:   root,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
		EventHandler: func(ev events.Event) {
			if ev.Type == events.TypeMemoryRecalled {
				recalled = true
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !recalled {
		t.Fatal("expected memory recall event")
	}
}

func TestMcpLocalToolName(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(fakeMCPTool{fakeTool: fakeTool{name: "docs"}, server: "ctx7"})
	if got := mcpLocalToolName(reg, "ctx7__docs"); got != "docs" {
		t.Fatalf("got %q", got)
	}
	if got := mcpLocalToolName(nil, "x"); got != "x" {
		t.Fatalf("nil reg: %q", got)
	}
}

func TestCompleteWithContinuesStreamNativeThinking(t *testing.T) {
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		return &provider.CompletionResponse{Content: "visible"}, nil
	}}
	var thinkChunks []string
	_, _, err := completeWithContinues(
		context.Background(), p,
		[]provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		nil,
		RunRequest{
			Goal: "q",
			StreamThinkDelta: func(delta string) error {
				thinkChunks = append(thinkChunks, delta)
				return nil
			},
			StreamThoughtStart: func(int) error { return nil },
		},
		1,
		continueConfig{},
		completeOptions{},
		func(events.Event) {},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = thinkChunks
}
