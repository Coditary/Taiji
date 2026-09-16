package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/provider"
)

const namePrompt = `Summarize the user's message as a short chat session title (3–6 words).
Reply with the title only — no quotes, no explanation.
Use the same language as the user message when possible.

User message:
`

// GenerateName asks the configured provider for a short session title.
func GenerateName(ctx context.Context, providers *provider.Registry, providerID, userMessage string) (string, error) {
	if providers == nil {
		return "", fmt.Errorf("provider registry is required")
	}
	userMessage = strings.TrimSpace(userMessage)
	if userMessage == "" {
		return "", fmt.Errorf("user message is required")
	}

	p, err := providers.Get(strings.TrimSpace(providerID))
	if err != nil {
		p, err = providers.Default()
		if err != nil {
			return "", err
		}
	}

	resp, err := p.Complete(ctx, provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: "You generate concise chat session titles."},
			{Role: provider.RoleUser, Content: namePrompt + userMessage},
		},
		MaxTokens:   40,
		Temperature: 0.3,
	})
	if err != nil {
		return fallbackName(userMessage), nil
	}

	name := sanitizeName(resp.Content)
	if name == "" {
		name = sanitizeName(provider.JoinThinkingBlocks(resp.ThinkingBlocks))
	}
	if name == "" {
		return fallbackName(userMessage), nil
	}
	return name, nil
}

func fallbackName(userMessage string) string {
	words := strings.Fields(strings.TrimSpace(userMessage))
	if len(words) == 0 {
		return "Chat"
	}
	if len(words) > 6 {
		words = words[:6]
	}
	return strings.Join(words, " ")
}

// AutoName sets an LLM-generated title when the session has none and was not manually named.
func AutoName(ctx context.Context, sess *Session, providers *provider.Registry, providerID, userMessage string) (string, error) {
	if sess == nil || !sess.NeedsAutoName() {
		return "", nil
	}
	name, err := GenerateName(ctx, providers, providerID, userMessage)
	if err != nil {
		return "", err
	}
	if err := sess.SetName(name, false); err != nil {
		return "", err
	}
	return name, nil
}
