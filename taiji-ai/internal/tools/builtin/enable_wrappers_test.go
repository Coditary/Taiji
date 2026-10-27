package builtin

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/decision"
	"github.com/coditary/taiji/internal/decisions"
	"github.com/coditary/taiji/internal/question"
	"github.com/coditary/taiji/internal/tools"
)

func TestEnableAskUserRegistersAndRun(t *testing.T) {
	reg := tools.NewRegistry()
	EnableAskUser(nil, question.NewService(question.CLIPrompter{}))
	EnableAskUser(reg, nil)
	svc := question.NewService(question.CLIPrompter{})
	EnableAskUser(reg, svc)
	if AskUserToolName() != "ask_user" {
		t.Fatal("tool name")
	}
	raw, ok := reg.Get("ask_user")
	if !ok {
		t.Fatal("missing ask_user")
	}
	tool, ok := raw.(AskUser)
	if !ok {
		t.Fatal("missing ask_user")
	}
	if tool.Name() == "" || tool.Description() == "" || tool.Parameters() == nil {
		t.Fatal("metadata")
	}
	svc.SetPrompter(stubAskPrompter{answers: []question.Answer{{"A"}}})
	if _, err := tool.Run(context.Background(), map[string]any{
		"questions": []any{map[string]any{
			"question": "Pick?",
			"options":  []any{map[string]any{"label": "A"}, map[string]any{"label": "B"}},
		}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEnableDecisionOrchestrationRegistersDecide(t *testing.T) {
	reg := tools.NewRegistry()
	EnableDecisionOrchestration(nil, nil, nil, "")
	svc := &decision.Service{Backend: stubDecisionBackend{}, DriverID: "laya", BackendName: "stub"}
	loader := decisions.NewLoader(nil)
	dir := t.TempDir()
	EnableDecisionOrchestration(reg, svc, loader, dir)
	raw, ok := reg.Get("decide")
	if !ok {
		t.Fatal("decide missing")
	}
	d, ok := raw.(Decide)
	if !ok {
		t.Fatal("decide missing")
	}
	if d.Name() != "decide" || d.Description() == "" || d.Parameters() == nil {
		t.Fatal("decide metadata")
	}
}

func TestDecisionRememberRunWrapper(t *testing.T) {
	dir := t.TempDir()
	tool := DecisionRemember{saveDir: dir, profiles: decisions.NewLoader([]string{dir})}
	if tool.Name() == "" || tool.Description() == "" || tool.Parameters() == nil {
		t.Fatal("metadata")
	}
	out, err := tool.Run(context.Background(), map[string]any{
		"name": "quick", "description": "d", "when": "w",
		"questions": map[string]any{"x": map[string]any{"type": "noul", "instructions": "?"}}})
	if err != nil || out == "" {
		t.Fatalf("err=%v out=%q", err, out)
	}
}

func TestEnableImageGenerationNilSafe(t *testing.T) {
	reg := tools.NewRegistry()
	EnableImageGeneration(nil, nil)
	EnableImageGeneration(reg, nil)
}
