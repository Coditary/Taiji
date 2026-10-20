package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools"
)

// EnableSkillOrchestration registers invoke_skill for on-demand skill loading.
func EnableSkillOrchestration(reg *tools.Registry, loader *skills.Loader) {
	if reg == nil || loader == nil {
		return
	}
	reg.Register(InvokeSkill{loader: loader})
}

// InvokeSkill loads a skill by invocation name.
type InvokeSkill struct {
	loader *skills.Loader
}

func (InvokeSkill) Name() string { return skills.InvokeSkillToolName }

func (InvokeSkill) Description() string {
	return "Load one or more skills by invocation name and return their full instructions. Multiple skills may be loaded in parallel in the same turn; stop and wait for all results before answering."
}

func (InvokeSkill) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Single skill name from the catalog (use hyphens, e.g. find-skills)",
			},
			"names": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Multiple skill names to load in one call",
			},
		},
	}
}

func (t InvokeSkill) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := t.RunWithMetadata(ctx, args)
	return res.Text, err
}

func (t InvokeSkill) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	names := skills.ParseInvokeNames(args)
	if len(names) == 0 {
		return tools.RunResult{}, fmt.Errorf("name or names is required")
	}
	all, err := t.loader.LoadAll()
	if err != nil {
		return tools.RunResult{}, err
	}
	parts := make([]string, 0, len(names))
	loaded := make([]string, 0, len(names))
	if rt, ok := session.RunContextFrom(ctx); ok {
		for _, name := range names {
			if !rt.SkillAllowed(name) {
				return tools.RunResult{}, fmt.Errorf("skill %q is not allowed in this chat", name)
			}
		}
	}
	for _, name := range names {
		skill, ok := skills.Find(all, name)
		if !ok {
			return tools.RunResult{}, fmt.Errorf("unknown skill %q", name)
		}
		parts = append(parts, skills.FormatInvoked(skill))
		loaded = append(loaded, skill.Name)
	}
	return tools.RunResult{
		Text: strings.Join(parts, "\n\n"),
		Data: map[string]any{
			"skill":  strings.Join(loaded, ", "),
			"skills": loaded,
		},
	}, nil
}
