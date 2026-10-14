package session

import "strings"

// MemoryReflectionUserPrompt nudges the model during the post-run memory pass.
const MemoryReflectionUserPrompt = "Store or revise memory for this run now."

// ToolUseNudgePrompt nudges the model to emit structured tool calls.
const ToolUseNudgePrompt = "Do not describe tools in prose. Call the appropriate tool now with valid JSON arguments. If you must use plain-text tool lines, put [tool call] at the very end of your message and write nothing after it."

// IsHiddenUserMessage reports synthetic user-role prompts that must not appear in TUI history.
func IsHiddenUserMessage(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	return trimmed == MemoryReflectionUserPrompt || trimmed == ToolUseNudgePrompt
}
