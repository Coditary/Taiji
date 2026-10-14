package builtin

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/question"
)

func TestAskUserRunWithMetadata(t *testing.T) {
	svc := question.NewService(question.CLIPrompter{})
	tool := AskUser{svc: svc}
	svc.SetPrompter(stubAskPrompter{answers: []question.Answer{{"Staging"}}})

	res, err := tool.RunWithMetadata(context.Background(), map[string]any{
		"questions": []any{
			map[string]any{
				"question": "Deploy where?",
				"options": []any{
					map[string]any{"label": "Staging"},
					map[string]any{"label": "Production"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunWithMetadata: %v", err)
	}
	if res.Text == "" {
		t.Fatal("expected formatted tool output")
	}
	if res.Data == nil {
		t.Fatal("expected structured data")
	}
}

type stubAskPrompter struct {
	answers []question.Answer
}

func (s stubAskPrompter) Prompt(context.Context, question.Request) ([]question.Answer, error) {
	return s.answers, nil
}
