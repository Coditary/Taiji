package session

import "strings"

// isRepeatUserPrompt reports a skill follow-up that repeats the same user text.
func isRepeatUserPrompt(pendingUser, content string) bool {
	pendingUser = strings.TrimSpace(pendingUser)
	content = strings.TrimSpace(content)
	return pendingUser != "" && content != "" && pendingUser == content
}
