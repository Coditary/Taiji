package skills

import "strings"

// ContinuationPrompt is a short system nudge after invoke_skill. Skill bodies live in the tool result only.
const ContinuationPrompt = "Skill instructions were returned by invoke_skill. Use them internally for the user's request now. Do not quote or repeat the skill text in your reply."

// legacyContinuationUserPrefix matches older sessions that duplicated skill text in a synthetic user message.
const legacyContinuationUserPrefix = "Skill instructions are loaded below."

// IsContinuationUserMessage reports legacy user-role skill follow-ups that must not appear in TUI history.
func IsContinuationUserMessage(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	if trimmed == ContinuationPrompt {
		return true
	}
	return strings.HasPrefix(trimmed, legacyContinuationUserPrefix)
}
