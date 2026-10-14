package agent

import (
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
)

func appendSkillContinuation(messages []provider.Message) []provider.Message {
	return append(messages, provider.Message{
		Role:    provider.RoleSystem,
		Content: skills.ContinuationPrompt,
	})
}
