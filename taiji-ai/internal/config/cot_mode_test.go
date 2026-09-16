package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestChainOfThoughtModeUnmarshal(t *testing.T) {
	cases := []struct {
		in   string
		want ChainOfThoughtMode
	}{
		{"prompt", ChainOfThoughtPrompt},
		{"off", ChainOfThoughtOff},
		{"full", ChainOfThoughtFull},
		{"true", ChainOfThoughtFull},
		{"false", ChainOfThoughtOff},
	}
	for _, tc := range cases {
		var got ChainOfThoughtMode
		if err := yaml.Unmarshal([]byte(tc.in), &got); err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDefaultChainOfThoughtIsOff(t *testing.T) {
	cfg := defaultConfig("/tmp")
	if cfg.Agent.ChainOfThought != ChainOfThoughtOff {
		t.Fatalf("default = %q, want off", cfg.Agent.ChainOfThought)
	}
	if cfg.Agent.SystemPrompt != "" {
		t.Fatal("default system prompt should be empty so mode-specific defaults apply")
	}
}

func TestChainOfThoughtModeSystemPromptAppendix(t *testing.T) {
	if ChainOfThoughtOff.SystemPromptAppendix() != "" {
		t.Fatal("off should not append")
	}
	if ChainOfThoughtPrompt.SystemPromptAppendix() != "" {
		t.Fatal("prompt should not append; instructions live in system_prompt")
	}
	if ChainOfThoughtFull.SystemPromptAppendix() == "" {
		t.Fatal("full should append")
	}
}
