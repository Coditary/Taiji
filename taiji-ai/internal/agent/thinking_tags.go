package agent

import "strings"

// Built via concatenation so tooling does not rewrite the literal tag names.
var (
	thinkingOpenTag  = "<" + "think" + ">"
	thinkingCloseTag = "</" + "think" + ">"
)

// normalizeThinkingTags maps common model-emitted aliases onto the canonical tags.
func normalizeThinkingTags(text string) string {
	replacements := []struct{ from, to string }{
		{"<" + "redacted_thinking" + ">", thinkingOpenTag},
		{"</" + "redacted_thinking" + ">", thinkingCloseTag},
		{"<" + "thinking" + ">", thinkingOpenTag},
		{"</" + "thinking" + ">", thinkingCloseTag},
	}
	for _, r := range replacements {
		text = strings.ReplaceAll(text, r.from, r.to)
	}
	return text
}

// sanitizeChainStream fixes common model formatting mistakes in chained CoT output.
func sanitizeChainStream(s string) string {
	s = normalizeThinkingTags(s)
	// Unfuse continue tokens glued directly to 
	for _, prefix := range []string{
		defaultContinueTrigger, "<|continue|>", "<|continue|", "<|continue", "|>", ">|",
	} {
		s = strings.ReplaceAll(s, prefix+thinkingOpenTag, "\n"+thinkingOpenTag)
	}
	for _, marker := range []string{defaultContinueTrigger, "<|continue|>", "<|continue|", "<|continue"} {
		s = strings.ReplaceAll(s, marker, "\n")
	}
	return s
}

func stripContinueMarkers(s string) string {
	for _, marker := range []string{defaultContinueTrigger, "<|continue|>", "<|continue|", "<|continue", "|>"} {
		s = strings.ReplaceAll(s, marker, "")
	}
	return s
}

// IsVisibleAnswerDelta reports whether streamed text should open the answer section.
func IsVisibleAnswerDelta(delta string) bool {
	peek := strings.TrimSpace(stripContinueMarkers(delta))
	return peek != "" && !isChainNoise(peek)
}

// isChainNoise reports text that must not be shown as a user-facing answer.
func isChainNoise(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if strings.Contains(s, thinkingOpenTag) || strings.Contains(s, thinkingCloseTag) {
		return true
	}
	if strings.Contains(s, "<|continue") {
		return true
	}
	if strings.TrimSpace(s) == "|>" {
		return true
	}
	return false
}
