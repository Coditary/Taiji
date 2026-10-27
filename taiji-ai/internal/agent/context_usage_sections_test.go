package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/skills"
)

func TestSystemPromptSectionsSkillsAndPresentation(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "demo", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte(`---
name: demo
description: Demo skill.
---
Body.`), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := skills.NewLoader([]string{dir})
	s := &session.Session{State: session.State{Settings: session.Settings{
		Presentation: "story",
	}}}
	rt := s.AgentRuntimeForRun(&taijicfg.Config{Agent: taijicfg.AgentConfig{
		ChainOfThought: taijicfg.ChainOfThoughtPrompt,
	}})

	core, _, skillsExtra := systemPromptSections(RunRequest{
		Session:     s,
		Skills:      loader,
		SkillNames:  []string{"demo"},
		Config:      &taijicfg.Config{Root: dir},
	}, rt, nil)
	if core == "" || !strings.Contains(core, "Spielleiter") {
		t.Fatalf("core missing presentation: len=%d", len(core))
	}
	if skillsExtra == "" {
		t.Fatal("expected skills appendix")
	}
}
