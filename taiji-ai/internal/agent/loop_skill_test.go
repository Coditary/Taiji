package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

type sequenceProvider struct {
	id        string
	responses []provider.CompletionResponse
	calls     int
}

func (p *sequenceProvider) ID() string   { return p.id }
func (p *sequenceProvider) Type() string { return "sequence" }

func (p *sequenceProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	if p.calls >= len(p.responses) {
		return &provider.CompletionResponse{Content: "fallback"}, nil
	}
	resp := p.responses[p.calls]
	p.calls++
	if req.OnDelta != nil && resp.Content != "" {
		if err := req.OnDelta(resp.Content); err != nil {
			return nil, err
		}
	}
	return &resp, nil
}

func TestRunContinuesAfterEmbeddedInvokeSkill(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hallo", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`---
name: hallo
description: Demo.
---

Say hi internally.`), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tools.NewRegistry()
	builtin.EnableSkillOrchestration(reg, skills.NewLoader([]string{dir}))

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: "Ich lade den Skill.\n[tool call] invoke_skill {\"name\": \"hallo\"}"},
			{Content: "Hallo! Skill befolgt."},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	result, err := Run(context.Background(), RunRequest{
		Goal:      "nutze nur den hallo skill",
		Providers: providers,
		Tools:     reg,
		Skills:    skills.NewLoader([]string{dir}),
		Config:    &taijicfg.Config{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq.calls != 2 {
		t.Fatalf("expected 2 provider calls, got %d", seq.calls)
	}
	if !strings.Contains(result.Answer, "Skill befolgt") {
		t.Fatalf("unexpected answer: %q", result.Answer)
	}
	foundToolSkill := false
	foundSystemNudge := false
	for _, m := range result.Messages {
		if m.Role == provider.RoleTool && strings.Contains(m.Content, "## Skill: hallo") {
			foundToolSkill = true
		}
		if m.Role == provider.RoleSystem && m.Content == skills.ContinuationPrompt {
			foundSystemNudge = true
		}
		if m.Role == provider.RoleUser && strings.Contains(m.Content, "## Skill: hallo") {
			t.Fatalf("skill body should not be duplicated in user message: %#v", m.Content)
		}
	}
	if !foundToolSkill {
		t.Fatal("expected invoke_skill tool result with skill body")
	}
	if !foundSystemNudge {
		t.Fatal("expected system continuation nudge after skill load")
	}
}
