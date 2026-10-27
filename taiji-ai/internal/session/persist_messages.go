package session

import (
	"strings"

	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
)

// SanitizeStoredMessages removes ephemeral prompt rows that must not appear in chat history.
func SanitizeStoredMessages(messages []provider.Message, settings Settings) []provider.Message {
	examples := fewShotContentSet(settings.FewShotExamples)
	out := make([]provider.Message, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case provider.RoleSystem:
			continue
		case provider.RoleUser:
			content := strings.TrimSpace(m.Content)
			if content == "" {
				continue
			}
			if examples[content] || strings.HasPrefix(content, "[Turn focus]") {
				continue
			}
			if skills.IsContinuationUserMessage(content) || IsHiddenUserMessage(content) {
				out = append(out, m)
				continue
			}
			out = append(out, m)
		case provider.RoleAssistant:
			content := strings.TrimSpace(m.Content)
			if examples[content] {
				continue
			}
			out = append(out, m)
		default:
			out = append(out, m)
		}
	}
	return out
}

// ApplyDisplayGoalToLastUser restores @-mentions and composer text in persisted history.
func ApplyDisplayGoalToLastUser(messages []provider.Message, displayGoal, modelGoal string) {
	displayGoal = strings.TrimSpace(displayGoal)
	modelGoal = strings.TrimSpace(modelGoal)
	if displayGoal == "" || displayGoal == modelGoal {
		return
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != provider.RoleUser {
			continue
		}
		content := strings.TrimSpace(messages[i].Content)
		if content == modelGoal || strings.HasSuffix(content, "\n\n"+modelGoal) || strings.HasSuffix(content, modelGoal) {
			messages[i].Content = displayGoal
		}
		return
	}
}

func fewShotContentSet(examples []ExampleTurn) map[string]bool {
	set := map[string]bool{}
	for _, ex := range examples {
		if c := strings.TrimSpace(ex.Content); c != "" {
			set[c] = true
		}
	}
	return set
}
