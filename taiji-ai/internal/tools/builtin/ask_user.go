package builtin

import (
	"context"
	"errors"

	"github.com/coditary/taiji/internal/question"
	"github.com/coditary/taiji/internal/tools"
)

const askUserToolName = "ask_user"

// EnableAskUser registers the ask_user tool when a question service is configured.
func EnableAskUser(reg *tools.Registry, svc *question.Service) {
	if reg == nil || svc == nil {
		return
	}
	reg.Register(AskUser{svc: svc})
}

// AskUserToolName returns the registered tool name.
func AskUserToolName() string { return askUserToolName }

// AskUser asks the human user structured questions during an agent run.
type AskUser struct {
	svc *question.Service
}

func (AskUser) Name() string { return askUserToolName }

func (AskUser) Description() string {
	return `Ask the user structured questions during execution. Use when you need preferences, clarifications, or implementation choices from the human.

Usage notes:
- Ask at most one ask_user call per assistant turn.
- Put the recommended option first and suffix its label with "(Recommended)" when applicable.
- When custom answers are enabled (default), do not add manual "Other" options.
- Answers are returned as arrays of labels per question; set multiple: true to allow multi-select.
- Use multiple questions in one call when several related choices belong together.`
}

func (AskUser) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"questions": map[string]any{
				"type":        "array",
				"description": "Questions to ask the user, shown as sequential panes.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Stable id for this question in the response payload",
						},
						"header": map[string]any{
							"type":        "string",
							"description": "Short pane title (max ~30 chars)",
						},
						"question": map[string]any{
							"type":        "string",
							"description": "Complete question shown to the user",
						},
						"options": map[string]any{
							"type":        "array",
							"description": "Selectable choices (minimum 2)",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"id": map[string]any{
										"type": "string",
									},
									"label": map[string]any{
										"type":        "string",
										"description": "Display text (keep concise)",
									},
									"description": map[string]any{
										"type":        "string",
										"description": "Optional explanation shown under the label",
									},
								},
								"required": []string{"label"},
							},
							"minItems": 2,
						},
						"multiple": map[string]any{
							"type":        "boolean",
							"description": "Allow selecting more than one option",
						},
						"custom": map[string]any{
							"type":        "boolean",
							"description": "Allow typing a custom answer (default: true)",
						},
					},
					"required": []string{"question", "options"},
				},
				"minItems": 1,
			},
		},
		"required": []string{"questions"},
	}
}

func (t AskUser) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := t.RunWithMetadata(ctx, args)
	return res.Text, err
}

func (t AskUser) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	if t.svc == nil {
		return tools.RunResult{}, question.ErrNoPrompter
	}

	questions, err := question.ParseQuestions(args)
	if err != nil {
		return tools.RunResult{}, err
	}

	var toolRef *question.ToolRef
	if callID := question.ToolCallIDFrom(ctx); callID != "" {
		toolRef = &question.ToolRef{CallID: callID}
	}

	answers, err := t.svc.Ask(ctx, question.AskInput{
		SessionID: question.SessionIDFrom(ctx),
		Questions: questions,
		Tool:      toolRef,
	})
	if err != nil {
		if errors.Is(err, question.ErrRejected) {
			return tools.RunResult{Text: "error: user dismissed the question"}, err
		}
		return tools.RunResult{}, err
	}

	data := question.AnswersData(questions, answers)
	data["questions"] = questions
	return tools.RunResult{
		Text: question.FormatToolOutput(questions, answers),
		Data: data,
	}, nil
}

func askUserQuestionsFromData(data map[string]any) []question.Info {
	if data == nil {
		return nil
	}
	raw, ok := data["questions"]
	if !ok {
		return nil
	}
	switch qs := raw.(type) {
	case []question.Info:
		return qs
	default:
		return nil
	}
}

// AskUserQuestionsFromResult extracts question infos from a tool result payload.
func AskUserQuestionsFromResult(data map[string]any) []question.Info {
	return askUserQuestionsFromData(data)
}
