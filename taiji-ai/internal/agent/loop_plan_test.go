package agent

import (
	"context"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func TestRunPlanPassFullChainOfThought(t *testing.T) {
	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: "Internal plan: step A, step B."},
			{Content: "Shipped answer."},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})
	cfg := &taijicfg.Config{Agent: taijicfg.AgentConfig{ChainOfThought: taijicfg.ChainOfThoughtFull}}

	result, err := Run(context.Background(), RunRequest{
		Goal:      "How would you implement a distributed cache?",
		Providers: providers,
		Tools:     tools.NewRegistry(),
		Config:    cfg,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq.calls < 2 {
		t.Fatalf("expected plan pass + answer, calls=%d", seq.calls)
	}
	if !strings.Contains(result.Answer, "Shipped answer") {
		t.Fatalf("answer=%q", result.Answer)
	}
}

func TestRunInteractionModePlan(t *testing.T) {
	seq := &sequenceProvider{
		id:        "seq",
		responses: []provider.CompletionResponse{{Content: "Plan only: step 1, step 2."}},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})
	s := &session.Session{State: session.State{Settings: session.Settings{InteractionMode: "plan"}}}

	result, err := Run(context.Background(), RunRequest{
		Goal:      "refactor the module",
		Providers: providers,
		Tools:     tools.NewRegistry(),
		Config:    &taijicfg.Config{},
		Session:   s,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Answer, "Plan only") {
		t.Fatalf("answer=%q", result.Answer)
	}
}
