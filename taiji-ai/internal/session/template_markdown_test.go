package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/config"
)

func TestLoadMarkdownTemplate(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, "global", "templates", "story.md")); err != nil {
		t.Skip("bundled md templates missing")
	}
	doc, err := LoadChatTemplate(root, "story")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Settings.Presentation != "narrative" {
		t.Fatalf("presentation = %q", doc.Settings.Presentation)
	}
	if len(doc.FewShotExamples) == 0 {
		t.Fatal("expected few-shot examples")
	}
	if len(doc.Settings.BuiltinToolsDeny) == 0 {
		t.Fatal("expected builtin_tools_deny")
	}
}

func TestApplyMarkdownTemplateFewShots(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(config.RuntimeDir(root), "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `---
id: demo
name: Demo
presentation: whatsapp
state_schema:
  mood: ok
---

Hello

<example>
{{user}}: Hi
{{actor}}: Hey
</example>
`
	if err := os.WriteFile(filepath.Join(dir, "demo.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, "1", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyChatTemplate(s, root, "demo"); err != nil {
		t.Fatal(err)
	}
	if len(s.State.Settings.FewShotExamples) != 2 {
		t.Fatalf("few shots = %d", len(s.State.Settings.FewShotExamples))
	}
	if s.State.Settings.ActorState["mood"] != "ok" {
		t.Fatalf("actor state = %#v", s.State.Settings.ActorState)
	}
}

func TestActorStateFromSchemaCoercesTypes(t *testing.T) {
	state := actorStateFromSchema(map[string]any{
		"flag": true,
		"count": 3,
		"ratio": 1.5,
		"note": " hi ",
	})
	if state["flag"] != "true" || state["count"] != "3" || state["ratio"] != "1.5" || state["note"] != "hi" {
		t.Fatalf("state=%v", state)
	}
	if actorStateFromSchema(nil) != nil {
		t.Fatal("nil schema")
	}
}
