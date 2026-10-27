package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/config"
)

func TestApplyChatTemplateMergesExtendedSettings(t *testing.T) {
	root := t.TempDir()
	tplDir := filepath.Join(config.RuntimeDir(root), "templates")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `id: rich
name: Rich
settings:
  provider: anthropic
  model: claude-test
  temperature: 0.2
  max_tokens: 4096
  max_steps: 12
  chain_of_thought: full
  thought_max_chars: 8000
  system_prompt_extra: extra rules
  skills_allow: [a]
  skills_deny: [b]
  default_packs: [pack-a]
  mcp_servers: [mcp-a]
  mcp_tools_deny: [srv:tool]
  presentation: story
  turn_metrics_preset: minimal
  tool_labels_locale: de
  preset: whatsapp
  decisions_enabled: false
  memory_enabled: false
  avatar: av.png
  theme_color: "#abc"
  template_display_name: Display
  builtin_tools_allow: [read_file]
  builtin_tools_deny: [bash]
  stream_pacing: natural
  council_enabled: true
  council_actors: [alice]
  follow_up_actor: bob
`
	if err := os.WriteFile(filepath.Join(tplDir, "rich.yaml"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, "tpl", "default-prov", "goal")
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyChatTemplate(s, root, "rich"); err != nil {
		t.Fatal(err)
	}
	st := s.State.Settings
	if st.Provider != "anthropic" || st.Model != "claude-test" {
		t.Fatalf("provider/model = %q / %q", st.Provider, st.Model)
	}
	if st.Temperature == nil || *st.Temperature != 0.2 {
		t.Fatalf("temperature = %#v", st.Temperature)
	}
	if st.DecisionsEnabled == nil || *st.DecisionsEnabled {
		t.Fatal("expected decisions disabled")
	}
	if len(st.PacksAllow) != 1 || st.PacksAllow[0] != "pack-a" {
		t.Fatalf("packs = %#v", st.PacksAllow)
	}
	if st.Presentation != "story" || st.StreamPacing != "natural" || !st.CouncilEnabled {
		t.Fatalf("ui settings = %+v", st)
	}
	if s.State.Provider != "anthropic" {
		t.Fatalf("session provider = %q", s.State.Provider)
	}
}
