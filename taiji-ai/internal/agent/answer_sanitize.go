package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/provider"
)

// stripLeadingToolDebris removes JSON/tool-call leftovers at the start of an answer.
func stripLeadingToolDebris(text string) string {
	trimmed := strings.TrimLeft(text, " \t\r\n")
	for {
		changed := false
		for _, prefix := range []string{`"}]`, `"]`, `"`, `}`, `]`, ","} {
			if strings.HasPrefix(trimmed, prefix) {
				trimmed = strings.TrimPrefix(trimmed, prefix)
				trimmed = strings.TrimLeft(trimmed, " \t\r\n")
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return trimmed
}

func shouldStripAnswerDebris(messages []provider.Message) bool {
	for i := len(messages) - 1; i >= 0; i-- {
		switch messages[i].Role {
		case provider.RoleTool:
			return true
		case provider.RoleAssistant:
			return false
		}
	}
	return false
}
