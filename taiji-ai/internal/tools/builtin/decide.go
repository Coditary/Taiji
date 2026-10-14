package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/decision"
	"github.com/coditary/taiji/internal/decisions"
	"github.com/coditary/taiji/internal/tools"
)

// EnableDecisionOrchestration registers decide and decision_remember when configured.
func EnableDecisionOrchestration(reg *tools.Registry, svc *decision.Service, profiles *decisions.Loader, saveDir string) {
	if reg == nil || svc == nil {
		return
	}
	reg.Register(Decide{svc: svc, profiles: profiles, saveDir: strings.TrimSpace(saveDir)})
	if strings.TrimSpace(saveDir) != "" {
		reg.Register(DecisionRemember{saveDir: saveDir, profiles: profiles})
	}
}

// Decide runs structured decision questions through the configured engine.
type Decide struct {
	svc      *decision.Service
	profiles *decisions.Loader
	saveDir  string
}

func (Decide) Name() string { return "decide" }

func (Decide) Description() string {
	return "Run structured decision questions (choice, noul, score) through the configured decision engine (e.g. laya). Use saved profiles via profile + text when one fits. When none fits, design custom questions inline. Optionally save reusable schemas with save_as or decision_remember."
}

func (Decide) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"profile": map[string]any{
				"type":        "string",
				"description": "Saved decision profile name (from the Saved decisions catalog)",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Alias for profile",
			},
			"decision": map[string]any{
				"type":        "string",
				"description": "Alias for profile",
			},
			"state": map[string]any{
				"type":        "string",
				"description": "Text or situation to evaluate",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Alias for state",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Optional model override for the decision engine",
			},
			"save_as": map[string]any{
				"type":        "string",
				"description": "When using custom questions, optionally save this schema as a reusable profile name (e.g. support-route)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Short summary when saving a new profile via save_as",
			},
			"when": map[string]any{
				"type":        "string",
				"description": "When to reuse this profile (used with save_as)",
			},
			"questions": map[string]any{
				"type":        "object",
				"description": "Custom on-demand questions when no saved profile fits. Map of question id to {type, instructions, criteria}.",
				"additionalProperties": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"type": map[string]any{
							"type": "string",
							"enum": []string{"choice", "noul", "score"},
						},
						"instructions": map[string]any{
							"type": "string",
						},
						"criteria": map[string]any{
							"type":        "array",
							"description": "Required for choice/score; option labels",
							"items":       map[string]any{"type": "string"},
						},
					},
					"required": []string{"type", "instructions"},
				},
			},
		},
	}
}

func (t Decide) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := t.RunWithMetadata(ctx, args)
	return res.Text, err
}

func (t Decide) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	if t.svc == nil {
		return tools.RunResult{}, decision.ErrDisabled
	}

	text := strings.TrimSpace(stringArg(args, "state"))
	if text == "" {
		text = strings.TrimSpace(stringArg(args, "text"))
	}
	if text == "" {
		return tools.RunResult{}, fmt.Errorf("state or text is required")
	}

	profileName := decisions.ParseProfileName(args)
	saveAs := strings.TrimSpace(stringArg(args, "save_as"))
	if saveAs != "" && profileName != "" {
		return tools.RunResult{}, fmt.Errorf("use either profile or save_as with custom questions, not both")
	}

	var questions []decision.Question
	var profileModel string
	var profileNameOut string
	customQuestions := false

	if profileName != "" {
		if t.profiles == nil {
			return tools.RunResult{}, fmt.Errorf("unknown decision profile %q (no profile directories configured)", profileName)
		}
		all, err := t.profiles.LoadAll()
		if err != nil {
			return tools.RunResult{}, err
		}
		profile, ok := decisions.Find(all, profileName)
		if !ok {
			return tools.RunResult{}, fmt.Errorf("unknown decision profile %q", profileName)
		}
		questions = append([]decision.Question(nil), profile.Questions...)
		profileModel = profile.Model
		profileNameOut = profile.Name
	} else {
		var err error
		questions, err = decision.ParseQuestions(args["questions"])
		if err != nil {
			return errResult(err)
		}
		customQuestions = true
	}

	model := strings.TrimSpace(stringArg(args, "model"))
	if model == "" {
		model = profileModel
	}

	resp, err := t.svc.Decide(ctx, decision.Request{
		Text:      text,
		Model:     model,
		Questions: questions,
	})
	if err != nil {
		return tools.RunResult{}, err
	}
	jsonText, err := decision.ToJSON(resp)
	if err != nil {
		return tools.RunResult{}, err
	}
	questionIDs := make([]string, 0, len(questions))
	for _, q := range questions {
		questionIDs = append(questionIDs, q.ID)
	}
	data := map[string]any{
		"response_json": jsonText,
		"backend":       resp.Backend,
		"driver":        resp.Driver,
		"model":         resp.Model,
		"state":         text,
		"questions":     questionIDs,
	}
	if profileNameOut != "" {
		data["profile"] = profileNameOut
	}

	textOut := decision.FormatText(resp)
	if saveAs != "" {
		if !customQuestions {
			return tools.RunResult{}, fmt.Errorf("save_as requires custom questions, not an existing profile")
		}
		path, saveErr := decisions.SaveProfile(t.saveDir, decisions.ProfileFromQuestions(
			saveAs,
			stringArg(args, "description"),
			stringArg(args, "when"),
			questions,
			"",
		))
		if saveErr != nil {
			return tools.RunResult{}, saveErr
		}
		data["saved_profile"] = saveAs
		data["saved_path"] = path
		textOut += fmt.Sprintf("\n\nSaved reusable decision profile %q at %s.", saveAs, path)
	}

	return tools.RunResult{
		Text: textOut,
		Data: data,
	}, nil
}

func errResult(err error) (tools.RunResult, error) {
	return tools.RunResult{}, err
}
