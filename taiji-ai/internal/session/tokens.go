package session

import (
	"strings"

	"github.com/coditary/taiji/internal/provider"
)

// EstimateTokens approximates token count from text length (~4 chars per token).
func EstimateTokens(text string) int {
	n := len([]rune(text))
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}

// EstimateMessagesTokens sums estimated tokens for chat history.
func EstimateMessagesTokens(messages []provider.Message) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m.Content)
		total += EstimateTokens(string(m.Role))
		if m.Name != "" {
			total += EstimateTokens(m.Name)
		}
		if m.ToolCallID != "" {
			total += EstimateTokens(m.ToolCallID)
		}
		// Per-message framing overhead.
		total += 4
	}
	return total
}

// contextWindowForModel guesses a model context limit from its name.
func contextWindowForModel(model string) int {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return 0
	}
	if strings.Contains(m, "200k") {
		return 200_000
	}
	if strings.Contains(m, "128k") {
		return 128_000
	}
	if strings.Contains(m, "64k") {
		return 64_000
	}
	if strings.Contains(m, "32k") {
		return 32_768
	}
	if strings.Contains(m, "16k") {
		return 16_384
	}
	if strings.Contains(m, "8k") {
		return 8_192
	}
	if strings.Contains(m, "claude") {
		return 200_000
	}
	if strings.Contains(m, "gpt-4") || strings.Contains(m, "gpt-5") || strings.Contains(m, "o1") || strings.Contains(m, "o3") {
		return 128_000
	}
	if strings.Contains(m, "gemini") {
		return 1_000_000
	}
	return 0
}
