package session

import (
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/graph"
)

func TestFormatActorStateSorted(t *testing.T) {
	got := FormatActorState(map[string]string{"mood": " calm ", "hp": "100"})
	if got != "hp: 100 · mood: calm" {
		t.Fatalf("got %q", got)
	}
	if FormatActorState(nil) != "" {
		t.Fatal("expected empty for nil map")
	}
}

func TestResolveProviderIDPrecedence(t *testing.T) {
	cfg := &taijicfg.Config{DefaultProvider: "project-default"}
	if got := (*Session)(nil).ResolveProviderID(cfg); got != "" {
		t.Fatalf("nil session: %q", got)
	}
	s := &Session{State: State{Provider: "session", Settings: Settings{Provider: "override"}}}
	if got := s.ResolveProviderID(cfg); got != "override" {
		t.Fatalf("settings override: %q", got)
	}
	s.State.Settings.Provider = ""
	if got := s.ResolveProviderID(cfg); got != "session" {
		t.Fatalf("session provider: %q", got)
	}
	s.State.Provider = ""
	if got := s.ResolveProviderID(cfg); got != "project-default" {
		t.Fatalf("project default: %q", got)
	}
}

func TestUISettingsForDisplayTurnMetricsPreset(t *testing.T) {
	cfg, err := taijicfg.LoadAt(t.TempDir(), taijicfg.LoadOptions{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	cfg.UI.TurnMetrics.Fields = []string{"latency"}
	s := &Session{State: State{Settings: Settings{TurnMetricsPreset: "minimal"}}}
	out := s.UISettingsForDisplay(cfg)
	if out.TurnMetrics.Preset != "minimal" || len(out.TurnMetrics.Fields) != 0 {
		t.Fatalf("turn metrics = %+v", out.TurnMetrics)
	}
	nilCfg := s.UISettingsForDisplay(nil)
	def := taijicfg.DefaultUIConfig()
	if nilCfg.KeybindingStyle != def.KeybindingStyle {
		t.Fatal("nil cfg should return defaults")
	}
}

func TestPresentationModeFallbacks(t *testing.T) {
	if got := (*Session)(nil).PresentationMode(); got != "markdown" {
		t.Fatalf("nil session: %q", got)
	}
	s := &Session{State: State{Settings: Settings{Preset: "story"}}}
	if s.PresentationMode() != "story" {
		t.Fatalf("preset fallback: %q", s.PresentationMode())
	}
	s.State.Settings.Presentation = "whatsapp"
	if s.PresentationMode() != "whatsapp" {
		t.Fatalf("presentation: %q", s.PresentationMode())
	}
}

func TestMCPToolRefParse(t *testing.T) {
	if got := MCPToolRef(" srv ", " tool "); got != "srv:tool" {
		t.Fatalf("ref = %q", got)
	}
	srv, tool, ok := ParseMCPToolRef("alpha:read_file")
	if !ok || srv != "alpha" || tool != "read_file" {
		t.Fatalf("parse = %q %q %v", srv, tool, ok)
	}
	if _, _, ok := ParseMCPToolRef("bad"); ok {
		t.Fatal("expected invalid ref")
	}
}

func TestAgentRuntimeForRunAddsGraphTools(t *testing.T) {
	bp := graph.MinimalBlueprint("g")
	s := &Session{State: State{
		Graph: &GraphSession{BlueprintID: "g", World: graph.NewWorldFromBlueprint(bp)},
		Settings: Settings{
			BuiltinToolsAllow: []string{"read_file"},
			ChatTools:         []ChatTool{{Name: "calc", Language: "python", Script: "print(1)"}},
		},
	}}
	rt := s.AgentRuntimeForRun(nil)
	names := strings.Join(rt.BuiltinToolsAllow, ",")
	for _, want := range GraphBuiltinToolNames() {
		if !strings.Contains(names, want) {
			t.Fatalf("missing graph tool %q in %q", want, names)
		}
	}
	if len(rt.ChatToolNames) != 1 || rt.ChatToolNames[0] != "calc" {
		t.Fatalf("chat tools = %#v", rt.ChatToolNames)
	}
}

func TestEnsureGraphToolsAllowedAppendsMissing(t *testing.T) {
	bp := graph.MinimalBlueprint("g")
	s := &Session{State: State{
		Graph:    &GraphSession{BlueprintID: "g", World: graph.NewWorldFromBlueprint(bp)},
		Settings: Settings{BuiltinToolsAllow: []string{"read_file"}},
	}}
	s.EnsureGraphToolsAllowed()
	allow := s.State.Settings.BuiltinToolsAllow
	if len(allow) < 4 {
		t.Fatalf("allow = %#v", allow)
	}
}
