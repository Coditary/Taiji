package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/decision"
	"github.com/coditary/taiji/internal/decisions"
	"github.com/coditary/taiji/internal/tools"
)

const DecisionRememberToolName = "decision_remember"

// DecisionRemember saves a reusable on-demand decision schema to .taiji/decisions/.
type DecisionRemember struct {
	saveDir  string
	profiles *decisions.Loader
}

func (DecisionRemember) Name() string { return DecisionRememberToolName }

func (DecisionRemember) Description() string {
	return "Save a custom decision schema for reuse. Use after designing questions for decide, or to persist an ad-hoc schema without re-running decide."
}

func (DecisionRemember) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Profile name (lowercase, hyphens), e.g. support-route",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Short summary of what this decision evaluates",
			},
			"when": map[string]any{
				"type":        "string",
				"description": "When to use this profile in future sessions",
			},
			"questions": map[string]any{
				"type":        "object",
				"description": "Question map: id -> {type, instructions, criteria?}",
				"additionalProperties": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"type": map[string]any{
							"type": "string",
							"enum": []string{"choice", "noul", "score"},
						},
						"instructions": map[string]any{"type": "string"},
						"criteria": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
					"required": []string{"type", "instructions"},
				},
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Optional markdown notes below the frontmatter",
			},
		},
		"required": []string{"name", "questions"},
	}
}

func (t DecisionRemember) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := t.RunWithMetadata(ctx, args)
	return res.Text, err
}

func (t DecisionRemember) RunWithMetadata(_ context.Context, args map[string]any) (tools.RunResult, error) {
	name, err := decisions.NormalizeProfileName(stringArg(args, "name"))
	if err != nil {
		return tools.RunResult{}, err
	}
	questions, err := decision.ParseQuestions(args["questions"])
	if err != nil {
		return tools.RunResult{}, err
	}
	if t.profiles != nil {
		all, loadErr := t.profiles.LoadAll()
		if loadErr != nil {
			return tools.RunResult{}, loadErr
		}
		if existing, ok := decisions.Find(all, name); ok && strings.TrimSpace(existing.Path) != "" {
			return tools.RunResult{}, fmt.Errorf("decision profile %q already exists at %s", name, existing.Path)
		}
	}

	profile := decisions.ProfileFromQuestions(
		name,
		stringArg(args, "description"),
		stringArg(args, "when"),
		questions,
		stringArg(args, "body"),
	)
	path, err := decisions.SaveProfile(t.saveDir, profile)
	if err != nil {
		return tools.RunResult{}, err
	}
	return tools.RunResult{
		Text: fmt.Sprintf("Saved decision profile %q at %s. Reuse with decide({\"profile\": %q, \"text\": \"…\"}).", name, path, name),
		Data: map[string]any{
			"profile": name,
			"path":    path,
		},
	}, nil
}
