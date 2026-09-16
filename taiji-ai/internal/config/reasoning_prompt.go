package config

// Method 1: strict tagged chain-of-thought appended to the system prompt.
func reasoningFormatPrompt() string {
	open := "<" + "think" + ">"
	close := "</" + "think" + ">"
	return `

Before responding to the user, you MUST reason inside ` + open + ` and ` + close + ` tags.
Inside those tags: analyze the request, plan your approach, and note possible pitfalls.
Write genuine step-by-step reasoning — not labels, bullets, or outline templates (no "Input:", "Context:", "Goal:").
After the closing tag, write only the final user-facing answer.
For trivial greetings or one-line requests, keep the thinking block brief.`
}

func reasoningPromptFor(mode ChainOfThoughtMode) string {
	switch mode.OrDefault() {
	case ChainOfThoughtFull:
		return reasoningFormatPrompt() + `
- On complex tasks, think thoroughly before answering.
- After a tool result, reflect briefly in a new thinking block before continuing.`
	case ChainOfThoughtPrompt:
		return ""
	default:
		return ""
	}
}

// ContinueSystemAppendix adds reasoning + auto-continue rules for full mode.
func ContinueSystemAppendix(trigger string) string {
	if trigger == "" {
		return ""
	}
	return reasoningFormatPrompt() + "\nWhen another reasoning step is needed, end with " + trigger + " on its own line. When reasoning is complete, omit that token."
}
