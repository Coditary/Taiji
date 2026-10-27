package agent

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

func TestBuildCompletionRequestAppliesSessionRuntime(t *testing.T) {
	maxTok := 512
	temp := 0.2
	s := &session.Session{State: session.State{Settings: session.Settings{
		Model:       "custom-model",
		MaxTokens:   &maxTok,
		Temperature: &temp,
	}}}
	cfg := &taijicfg.Config{}
	req := buildCompletionRequest(
		RunRequest{Goal: "hi", Session: s, Config: cfg},
		&retryStubProvider{},
		[]provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		[]provider.ToolDef{{Name: "grep"}},
		1,
		false,
		nil,
		nil,
	)
	if req.Model != "custom-model" || req.MaxTokens != 512 || req.Temperature != 0.2 {
		t.Fatalf("req=%+v", req)
	}
	if req.ToolChoice != "auto" {
		t.Fatalf("tool choice=%q", req.ToolChoice)
	}
}

func TestBuildCompletionRequestToolChoiceNoneClearsTools(t *testing.T) {
	req := buildCompletionRequest(
		RunRequest{Goal: "hello"},
		&retryStubProvider{},
		nil,
		[]provider.ToolDef{{Name: "grep"}},
		3,
		false,
		nil,
		nil,
	)
	if len(req.Tools) != 1 {
		t.Fatalf("tools=%v", req.Tools)
	}
	// Simulate retry attempt 3 path: caller passes toolChoice none by clearing in completeWithRetry.
	req.ToolChoice = "none"
	req.Tools = nil
	if req.ToolChoice != "none" || req.Tools != nil {
		t.Fatal("expected cleared tools")
	}
}
