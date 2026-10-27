package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestRunRequiresProviderRegistry(t *testing.T) {
	_, err := Run(context.Background(), RunRequest{Tools: tools.NewRegistry()})
	if err == nil || !strings.Contains(err.Error(), "provider registry") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunRequiresToolRegistry(t *testing.T) {
	reg := testDummyRegistry(t)
	_, err := Run(context.Background(), RunRequest{Providers: reg})
	if err == nil || !strings.Contains(err.Error(), "tool registry") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunMultiTurnEmbeddedReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"path": path})

	reg := tools.NewRegistry()
	reg.Register(builtin.ReadFile{})

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: fmt.Sprintf("Reading file.\n[tool call] read_file %s", string(args))},
			{Content: "Done reading."},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	result, err := Run(context.Background(), RunRequest{
		Goal:       "read the note",
		Providers:  providers,
		Tools:      reg,
		Config:     &taijicfg.Config{},
		WorkDir:    dir,
		SubRun:     SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq.calls != 2 {
		t.Fatalf("provider calls=%d want 2", seq.calls)
	}
	if !strings.Contains(result.Answer, "Done reading") {
		t.Fatalf("answer=%q", result.Answer)
	}
	foundTool := false
	for _, m := range result.Messages {
		if m.Role == provider.RoleTool && m.Name == "read_file" && strings.Contains(m.Content, "payload") {
			foundTool = true
		}
	}
	if !foundTool {
		t.Fatal("expected read_file observation with payload")
	}
}

func TestRunNativeToolCallsMultiTurn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(path, []byte("native"), 0o644); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"path": path})

	reg := tools.NewRegistry()
	reg.Register(builtin.ReadFile{})

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{ToolCalls: []provider.ToolCall{{ID: "tc1", Name: "read_file", Args: string(args)}}},
			{Content: "native path complete"},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	result, err := Run(context.Background(), RunRequest{
		Goal:      "open data.txt",
		Providers: providers,
		Tools:     reg,
		Config:    &taijicfg.Config{},
		WorkDir:   dir,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq.calls != 2 {
		t.Fatalf("calls=%d", seq.calls)
	}
	if result.Answer != "native path complete" {
		t.Fatalf("answer=%q", result.Answer)
	}
}

func TestRunEmitsToolCallAndObservationEvents(t *testing.T) {
	dir := t.TempDir()
	reg := tools.NewRegistry()
	reg.Register(builtin.ListDir{})

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: `[tool call] list_dir {"path": "."}`},
			{Content: "listed"},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	var types []events.Type
	_, err := Run(context.Background(), RunRequest{
		Goal:      "list workspace",
		Providers: providers,
		Tools:     reg,
		Config:    &taijicfg.Config{},
		WorkDir:   dir,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
		EventHandler: func(ev events.Event) {
			types = append(types, ev.Type)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	hasCall, hasObs, hasDone := false, false, false
	for _, ty := range types {
		switch ty {
		case events.TypeToolCall:
			hasCall = true
		case events.TypeObservation:
			hasObs = true
		case events.TypeDone:
			hasDone = true
		}
	}
	if !hasCall || !hasObs || !hasDone {
		t.Fatalf("events=%v", types)
	}
}

func TestRunMaxStepsReachedAfterToolRound(t *testing.T) {
	dir := t.TempDir()
	reg := tools.NewRegistry()
	reg.Register(builtin.ListDir{})

	max := 1
	s, err := session.New(dir, "max-steps", "seq", "chat")
	if err != nil {
		t.Fatal(err)
	}
	s.State.Settings.MaxSteps = &max

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: `[tool call] list_dir {"path": "."}`},
			{Content: "never reached"},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	_, err = Run(context.Background(), RunRequest{
		Goal:      "keep listing",
		Providers: providers,
		Tools:     reg,
		Config:    &taijicfg.Config{},
		WorkDir:   dir,
		Session:   s,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err == nil || !strings.Contains(err.Error(), "max steps") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunProviderCompleteErrorEmitsError(t *testing.T) {
	fail := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		return nil, errors.New("provider down")
	}}
	providers := provider.NewStaticRegistry("fail", map[string]provider.Provider{"fail": fail})

	var sawError bool
	_, err := Run(context.Background(), RunRequest{
		Goal:      "hello",
		ProviderID: "fail",
		Providers: providers,
		Tools:     tools.NewRegistry(),
		Config:    &taijicfg.Config{},
		SubRun:    SubRunConfig{SkipTurnRouter: true},
		EventHandler: func(ev events.Event) {
			if ev.Type == events.TypeError {
				sawError = true
			}
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !sawError {
		t.Fatal("expected TypeError event")
	}
}

func TestRunSessionStartHookError(t *testing.T) {
	hooks.RegisterPlugin("agentTestSessErr", func(ctx context.Context, input hooks.ShellInput) (hooks.Result, error) {
		return hooks.Result{}, errors.New("blocked at start")
	})
	runner := hooks.NewRunner([]hooks.Entry{{
		ID: "sess-err", Event: hooks.EventSessionStart, Actions: []hooks.ActionSpec{{Plugin: "agentTestSessErr"}},
	}})

	_, err := Run(context.Background(), RunRequest{
		Goal:      "hi",
		Providers: testDummyRegistry(t),
		Tools:     tools.NewRegistry(),
		Config:    &taijicfg.Config{},
		Hooks:     runner,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunContextBeforeHookError(t *testing.T) {
	hooks.RegisterPlugin("agentTestCtxErr", func(ctx context.Context, input hooks.ShellInput) (hooks.Result, error) {
		return hooks.Result{}, errors.New("context hook failed")
	})
	runner := hooks.NewRunner([]hooks.Entry{{
		ID: "ctx-err", Event: hooks.EventContextBefore, Actions: []hooks.ActionSpec{{Plugin: "agentTestCtxErr"}},
	}})

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{{Content: "ok"}},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	_, err := Run(context.Background(), RunRequest{
		Goal:      "hi",
		Providers: providers,
		Tools:     tools.NewRegistry(),
		Config:    &taijicfg.Config{},
		Hooks:     runner,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err == nil || !strings.Contains(err.Error(), "context hook") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunPersistsSessionMessages(t *testing.T) {
	root := t.TempDir()
	s, err := session.New(root, "persist", "seq", "chat")
	if err != nil {
		t.Fatal(err)
	}
	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{{Content: "saved answer"}},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	_, err = Run(context.Background(), RunRequest{
		Goal:       "remember this",
		DisplayGoal: "remember this",
		Providers:  providers,
		Tools:      tools.NewRegistry(),
		Config:     &taijicfg.Config{},
		Session:    s,
		SubRun:     SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.State.Messages) < 2 {
		t.Fatalf("messages=%d", len(s.State.Messages))
	}
	last := s.State.Messages[len(s.State.Messages)-1]
	if last.Role != provider.RoleAssistant || !strings.Contains(last.Content, "saved answer") {
		t.Fatalf("last message=%+v", last)
	}
}

func TestDelegateTaskRunWithDummyProvider(t *testing.T) {
	root := t.TempDir()
	reg := tools.NewRegistry()
	prov := testDummyRegistry(t)
	main := tools.NewRegistry()
	EnableDelegateTask(reg, DelegateTaskConfig{
		Providers: prov,
		MainTools: main,
		Config:    &taijicfg.Config{},
		WorkDir:   root,
	})
	tool, ok := reg.Get("delegate_task")
	if !ok {
		t.Fatal("missing delegate_task")
	}
	out, err := tool.Run(context.Background(), map[string]any{"prompt": "Ping"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Pong" {
		t.Fatalf("out=%q", out)
	}
}

func TestDelegateTaskRunCustomToolsAllow(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	main := tools.NewRegistry()
	main.Register(builtin.ReadFile{})
	reg := tools.NewRegistry()
	prov := testDummyRegistry(t)
	EnableDelegateTask(reg, DelegateTaskConfig{
		Providers: prov,
		MainTools: main,
		Config:    &taijicfg.Config{},
		WorkDir:   root,
	})
	tool, _ := reg.Get("delegate_task")
	// Dummy echoes prompt; sub-agent still runs without error.
	out, err := tool.Run(context.Background(), map[string]any{
		"prompt":      "Ping",
		"tools_allow": []any{"read_file"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Pong" {
		t.Fatalf("out=%q", out)
	}
}
