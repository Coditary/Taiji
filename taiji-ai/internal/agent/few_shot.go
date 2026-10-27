package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

func injectFewShotExamples(messages []provider.Message, sess *session.Session) []provider.Message {
	if sess == nil || len(messages) == 0 {
		return messages
	}
	examples := sess.State.Settings.FewShotExamples
	if len(examples) == 0 {
		return messages
	}
	out := make([]provider.Message, 0, len(messages)+len(examples))
	out = append(out, messages[0])
	for _, ex := range examples {
		role := provider.RoleUser
		if ex.Role == "assistant" {
			role = provider.RoleAssistant
		}
		if strings.TrimSpace(ex.Content) == "" {
			continue
		}
		out = append(out, provider.Message{Role: role, Content: ex.Content})
	}
	out = append(out, messages[1:]...)
	return out
}
