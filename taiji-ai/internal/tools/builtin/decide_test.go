package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/decision"
	"github.com/coditary/taiji/internal/decisions"
)

type stubDecisionBackend struct{}

func (stubDecisionBackend) Decide(_ context.Context, req decision.Request) (*decision.Response, error) {
	return &decision.Response{
		Backend: "stub",
		Driver:  "laya",
		Answers: map[string]decision.Answer{
			req.Questions[0].ID: {ID: req.Questions[0].ID, Type: decision.QuestionNoul, Noul: &decision.NoulAnswer{Value: 0.8}},
		},
	}, nil
}

func TestDecideUsesSavedProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "buy", "DECISION.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `---
name: buy
description: Purchase decision
questions:
  worth_it:
    type: noul
    instructions: "Is it worth buying?"
---
`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &decision.Service{Backend: stubDecisionBackend{}, DriverID: "laya", BackendName: "stub"}
	loader := decisions.NewLoader([]string{dir})
	tool := Decide{svc: svc, profiles: loader}

	res, err := tool.RunWithMetadata(context.Background(), map[string]any{
		"profile": "buy",
		"text":    "Laptop 1200 EUR",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Data["profile"] != "buy" {
		t.Fatalf("profile data = %#v", res.Data["profile"])
	}
	if res.Text == "" {
		t.Fatal("expected formatted text")
	}
}

func TestDecideCustomQuestionsSaveAs(t *testing.T) {
	dir := t.TempDir()
	svc := &decision.Service{Backend: stubDecisionBackend{}, DriverID: "laya", BackendName: "stub"}
	tool := Decide{svc: svc, saveDir: dir}

	res, err := tool.RunWithMetadata(context.Background(), map[string]any{
		"text": "Help! My payouts failed.",
		"save_as": "support-route",
		"description": "Route support tickets",
		"questions": map[string]any{
			"route": map[string]any{
				"type":         "choice",
				"instructions": "Which team?",
				"criteria":     []any{"billing", "technical"},
			},
			"urgent": map[string]any{
				"type":         "noul",
				"instructions": "Is it urgent?",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Data["saved_profile"] != "support-route" {
		t.Fatalf("saved_profile = %#v", res.Data["saved_profile"])
	}
	if _, err := os.Stat(filepath.Join(dir, "support-route", "DECISION.md")); err != nil {
		t.Fatal(err)
	}
}

func TestDecideUnknownProfile(t *testing.T) {
	svc := &decision.Service{Backend: stubDecisionBackend{}, DriverID: "laya", BackendName: "stub"}
	tool := Decide{svc: svc, profiles: decisions.NewLoader([]string{t.TempDir()})}
	_, err := tool.Run(context.Background(), map[string]any{
		"profile": "missing",
		"text":    "test",
	})
	if err == nil {
		t.Fatal("expected unknown profile error")
	}
}
