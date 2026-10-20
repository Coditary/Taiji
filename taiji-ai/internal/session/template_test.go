package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBundledWhatsAppTemplate(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, "global", "templates", "whatsapp.yaml")); err != nil {
		t.Skip("bundled templates not in workspace layout")
	}
	doc, err := LoadChatTemplate(root, "whatsapp")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Settings.Preset != "whatsapp" {
		t.Fatalf("preset = %q", doc.Settings.Preset)
	}
}

func TestApplyChatTemplate(t *testing.T) {
	root := t.TempDir()
	tplDir := filepath.Join(root, ".taiji", "templates")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `id: test
name: Test
settings:
  max_steps: 7
  mcp_servers: [alpha]
  skills_allow: [skill-a]
`
	if err := os.WriteFile(filepath.Join(tplDir, "test.yaml"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, "1", "p", "goal")
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyChatTemplate(s, root, "test"); err != nil {
		t.Fatal(err)
	}
	if s.State.Settings.TemplateID != "test" {
		t.Fatalf("template id = %q", s.State.Settings.TemplateID)
	}
	rt := s.AgentRuntimeForRun(nil)
	if rt.MaxSteps != 7 {
		t.Fatalf("max_steps = %d", rt.MaxSteps)
	}
	if len(rt.MCPServers) != 1 || rt.MCPServers[0] != "alpha" {
		t.Fatalf("mcp = %#v", rt.MCPServers)
	}
	if !rt.SkillAllowed("skill-a") || rt.SkillAllowed("skill-b") {
		t.Fatalf("skills allow mismatch")
	}
}
