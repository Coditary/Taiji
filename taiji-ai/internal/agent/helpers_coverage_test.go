package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/packs"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestPresentationPromptAppendix(t *testing.T) {
	for preset, wantSub := range map[string]string{
		"whatsapp": "Messenger",
		"story":    "Spielleiter",
		"minimal":  "knapp",
		"":         "",
		"unknown":  "",
	} {
		got := presentationPromptAppendix(preset)
		if wantSub == "" {
			if got != "" {
				t.Fatalf("preset %q: got %q", preset, got)
			}
			continue
		}
		if !strings.Contains(got, wantSub) {
			t.Fatalf("preset %q: got %q", preset, got)
		}
	}
}

func TestSessionPresentationPreset(t *testing.T) {
	if sessionPresentationPreset(nil) != "" {
		t.Fatal("nil session")
	}
	s := &session.Session{State: session.State{Settings: session.Settings{Presentation: "story"}}}
	if sessionPresentationPreset(s) != "story" {
		t.Fatalf("got %q", sessionPresentationPreset(s))
	}
}

func TestInjectFewShotExamples(t *testing.T) {
	base := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "goal"},
	}
	if got := injectFewShotExamples(nil, &session.Session{}); got != nil {
		t.Fatal("nil messages unchanged")
	}
	if got := injectFewShotExamples(base, nil); len(got) != len(base) {
		t.Fatal("nil session unchanged")
	}
	s := &session.Session{State: session.State{Settings: session.Settings{
		FewShotExamples: []session.ExampleTurn{
			{Role: "user", Content: "ex user"},
			{Role: "assistant", Content: "ex assist"},
			{Role: "user", Content: "   "},
		},
	}}}
	out := injectFewShotExamples(base, s)
	if len(out) != 4 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Content != "sys" || out[1].Content != "ex user" || out[2].Content != "ex assist" || out[3].Content != "goal" {
		t.Fatalf("order wrong: %#v", out)
	}
}

func TestFilterProviderToolsRespectsAllowDeny(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(builtin.ListDir{})
	reg.Register(builtin.ReadFile{})
	s := &session.Session{State: session.State{Settings: session.Settings{
		BuiltinToolsAllow: []string{"list_dir"},
	}}}
	rt := s.AgentRuntimeForRun(&taijicfg.Config{})
	defs := filterProviderTools(reg, rt)
	if len(defs) != 1 || defs[0].Name != "list_dir" {
		t.Fatalf("defs=%#v", defs)
	}
}

func TestApplyInteractionModeToolsPlan(t *testing.T) {
	in := []provider.ToolDef{{Name: "grep"}, {Name: "switch_agent_mode"}, {Name: "read_file"}}
	out := applyInteractionModeTools(in, "plan")
	if len(out) != 1 || out[0].Name != "switch_agent_mode" {
		t.Fatalf("plan mode: %#v", out)
	}
	if got := applyInteractionModeTools(in, "agent"); len(got) != 3 {
		t.Fatalf("agent mode: len=%d", len(got))
	}
}

func TestSessionInteractionMode(t *testing.T) {
	s := &session.Session{State: session.State{Settings: session.Settings{InteractionMode: "plan"}}}
	rt := session.AgentRuntime{InteractionMode: "agent"}
	if sessionInteractionMode(s, rt) != "plan" {
		t.Fatal("session override")
	}
	if sessionInteractionMode(nil, rt) != "agent" {
		t.Fatal("runtime fallback")
	}
}

func TestEstimateContextUsage(t *testing.T) {
	root := t.TempDir()
	s, err := session.New(root, "s1", "dummy", "hello")
	if err != nil {
		t.Fatal(err)
	}
	s.State.Messages = []provider.Message{
		{Role: provider.RoleUser, Content: "Was ist los?"},
		{Role: provider.RoleAssistant, Content: "Alles gut."},
	}
	reg := tools.NewRegistry()
	reg.Register(builtin.ListDir{})
	cfg := &taijicfg.Config{Root: root, Agent: taijicfg.AgentConfig{SystemPrompt: "Custom system."}}
	br := EstimateContextUsage(RunRequest{
		Session: s,
		Config:  cfg,
		Tools:   reg,
	}, nil)
	if br.SystemTokens <= 0 {
		t.Fatalf("system tokens=%d", br.SystemTokens)
	}
	if br.ConversationTokens <= 0 {
		t.Fatalf("conversation tokens=%d", br.ConversationTokens)
	}
	if br.ToolsTokens <= 0 {
		t.Fatalf("tools tokens=%d", br.ToolsTokens)
	}
}

func TestRecentUserTextAndPackAllowList(t *testing.T) {
	if recentUserText(nil, 2) != "" {
		t.Fatal("nil session")
	}
	s := &session.Session{State: session.State{Messages: []provider.Message{
		{Role: provider.RoleUser, Content: "first"},
		{Role: provider.RoleAssistant, Content: "ok"},
		{Role: provider.RoleUser, Content: "second"},
	}}}
	if got := recentUserText(s, 1); got != "second" {
		t.Fatalf("got %q", got)
	}
	cfg := &taijicfg.Config{}
	s.State.Settings.PacksAllow = []string{"lore"}
	if allow := packAllowList(RunRequest{Session: s, Config: cfg}); len(allow) != 1 {
		t.Fatalf("allow=%v", allow)
	}
	if packAllowList(RunRequest{}) != nil {
		t.Fatal("no session")
	}
}

func TestRunPackAssemblyEmitsSummary(t *testing.T) {
	root := t.TempDir()
	packRoot := filepath.Join(root, ".taiji", "packs", "lore")
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "rules.md"), []byte(`---
injection: always
---
Pack body.`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &taijicfg.Config{Root: root, Packs: taijicfg.PacksConfig{Dirs: []string{filepath.Join(root, ".taiji", "packs")}}}
	idx := packs.NewIndex(packs.NewLoader(cfg.Packs.Dirs), nil, cfg, root)
	if err := idx.Reload(); err != nil {
		t.Fatal(err)
	}
	var seen events.Type
	res := runPackAssembly(context.Background(), RunRequest{
		PackIndex: idx,
		Session:   &session.Session{},
	}, "hello", func(ev events.Event) {
		if ev.Type == events.TypeObservation {
			seen = ev.Type
		}
	})
	if res.systemAppendix == "" {
		t.Fatal("expected appendix")
	}
	if seen != events.TypeObservation {
		t.Fatalf("emit type=%q", seen)
	}
}

func TestParseToolsAllow(t *testing.T) {
	if len(parseToolsAllow(map[string]any{})) != len(defaultDelegateToolsAllow) {
		t.Fatal("default")
	}
	custom := parseToolsAllow(map[string]any{"tools_allow": []any{"grep", "  ", "read_file"}})
	if len(custom) != 2 || custom[0] != "grep" {
		t.Fatalf("custom=%v", custom)
	}
	if len(parseToolsAllow(map[string]any{"tools_allow": []string{}})) != len(defaultDelegateToolsAllow) {
		t.Fatal("empty slice")
	}
}

func TestEnableDelegateTaskRegistersTool(t *testing.T) {
	reg := tools.NewRegistry()
	EnableDelegateTask(reg, DelegateTaskConfig{})
	if _, ok := reg.Get("delegate_task"); ok {
		t.Fatal("incomplete config should not register")
	}
	prov := testDummyRegistry(t)
	main := tools.NewRegistry()
	EnableDelegateTask(reg, DelegateTaskConfig{Providers: prov, MainTools: main, Config: &taijicfg.Config{}})
	tool, ok := reg.Get("delegate_task")
	if !ok {
		t.Fatal("missing delegate_task")
	}
	if tool.Name() != "delegate_task" || tool.Description() == "" || tool.Parameters() == nil {
		t.Fatal("metadata")
	}
}

func TestDelegateTaskRunRequiresPrompt(t *testing.T) {
	reg := tools.NewRegistry()
	prov := testDummyRegistry(t)
	main := tools.NewRegistry()
	EnableDelegateTask(reg, DelegateTaskConfig{Providers: prov, MainTools: main, Config: &taijicfg.Config{}})
	tool, _ := reg.Get("delegate_task")
	if _, err := tool.Run(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error")
	}
}
