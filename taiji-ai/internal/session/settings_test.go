package session

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestAgentRuntimeForRunSessionOverrides(t *testing.T) {
	cfg, err := taijicfg.LoadAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Agent.MaxSteps = 10
	cfg.Agent.ChainOfThought = taijicfg.ChainOfThoughtOff

	s := &Session{
		State: State{
			Provider: "local",
			Settings: Settings{
				MaxSteps:       intPtr(40),
				ChainOfThought: "full",
				Model:          "custom-model",
			},
		},
	}
	rt := s.AgentRuntimeForRun(cfg)
	if rt.MaxSteps != 40 {
		t.Fatalf("max_steps = %d", rt.MaxSteps)
	}
	if rt.ChainOfThought != taijicfg.ChainOfThoughtFull {
		t.Fatalf("cot = %q", rt.ChainOfThought)
	}
	if rt.Model != "custom-model" {
		t.Fatalf("model = %q", rt.Model)
	}
}

func TestSkillAllowed(t *testing.T) {
	rt := AgentRuntime{SkillsAllow: []string{"a"}, SkillsDeny: []string{"b"}}
	if !rt.SkillAllowed("a") || rt.SkillAllowed("b") || rt.SkillAllowed("c") {
		t.Fatalf("allowlist mismatch")
	}
}

func intPtr(v int) *int { return &v }
