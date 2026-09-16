package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ChainOfThoughtMode controls how the agent applies reasoning instructions.
//
//   - off: no extra reasoning instructions (default)
//   - prompt: tagged chain-of-thought system prompt; one model call per turn
//   - full: explicit planning pass + auto-continue for complex tasks
type ChainOfThoughtMode string

const (
	ChainOfThoughtOff    ChainOfThoughtMode = "off"
	ChainOfThoughtPrompt ChainOfThoughtMode = "prompt"
	ChainOfThoughtFull   ChainOfThoughtMode = "full"
)

// OrDefault returns off when unset.
func (m ChainOfThoughtMode) OrDefault() ChainOfThoughtMode {
	if m == "" {
		return ChainOfThoughtOff
	}
	return m
}

// UsesFullPass reports whether the multi-turn planning pass may run.
func (m ChainOfThoughtMode) UsesFullPass() bool {
	return m.OrDefault() == ChainOfThoughtFull
}

// SystemPromptAppendix returns extra system text for this mode, or "" for off.
func (m ChainOfThoughtMode) SystemPromptAppendix() string {
	return reasoningPromptFor(m)
}

func (m *ChainOfThoughtMode) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err == nil {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "", "prompt":
			*m = ChainOfThoughtPrompt
		case "off", "false", "none", "0":
			*m = ChainOfThoughtOff
		case "full", "true", "1":
			*m = ChainOfThoughtFull
		default:
			return fmt.Errorf("invalid chain_of_thought %q (use off, prompt, or full)", s)
		}
		return nil
	}
	var b bool
	if err := value.Decode(&b); err == nil {
		if b {
			*m = ChainOfThoughtFull
		} else {
			*m = ChainOfThoughtOff
		}
		return nil
	}
	return fmt.Errorf("chain_of_thought must be off, prompt, full, or a boolean")
}

func (m ChainOfThoughtMode) MarshalYAML() (interface{}, error) {
	switch m.OrDefault() {
	case ChainOfThoughtOff:
		return "off", nil
	case ChainOfThoughtFull:
		return "full", nil
	default:
		return "prompt", nil
	}
}
