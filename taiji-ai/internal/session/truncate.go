package session

import (
	"strings"

	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
)

// TruncateAfterTurn keeps messages through the user message at turnIndex and drops
// that turn's assistant reply plus everything after it. Tool and hidden messages
// before the cut are preserved.
func TruncateAfterTurn(messages []provider.Message, turnIndex int) []provider.Message {
	if turnIndex < 0 || len(messages) == 0 {
		return messages
	}

	turn := -1
	pendingUser := false
	pendingUserStart := -1

	for i, msg := range messages {
		switch msg.Role {
		case provider.RoleUser:
			content := strings.TrimSpace(msg.Content)
			if skills.IsContinuationUserMessage(content) || IsHiddenUserMessage(content) {
				continue
			}
			if IsCompactUserMessage(content) {
				if pendingUser {
					turn++
					if turn > turnIndex {
						return messages[:pendingUserStart]
					}
					pendingUser = false
				}
				turn++
				if turn > turnIndex {
					return messages[:i]
				}
				continue
			}
			if pendingUser && isRepeatUserPrompt(messages[pendingUserStart].Content, content) {
				continue
			}
			if pendingUser {
				turn++
				if turn > turnIndex {
					return messages[:pendingUserStart]
				}
			}
			turn++
			pendingUserStart = i
			pendingUser = true
			if turn > turnIndex {
				return messages[:i]
			}
		case provider.RoleAssistant:
			content := strings.TrimSpace(msg.Content)
			if !pendingUser || content == "" {
				continue
			}
			if turn == turnIndex {
				return messages[:i]
			}
			pendingUser = false
		}
	}

	if turn == turnIndex && pendingUser {
		return messages
	}
	return messages
}

// VisibleTurnCount matches TurnsFromMessages length.
func VisibleTurnCount(messages []provider.Message) int {
	return len(TurnsFromMessages(messages))
}
