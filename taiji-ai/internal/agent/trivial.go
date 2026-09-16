package agent

import (
	"strings"
)

// needsDeepReasoning decides whether to run the full planning pass and auto-continue loop.
// Only used when chain_of_thought is "full".
func needsDeepReasoning(goal string) bool {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return false
	}

	lower := strings.ToLower(goal)

	for _, kw := range []string{
		"ohne nachzudenken", "ohne zu überlegen", "ohne zu ueberlegen",
		"direkt antwort", "kurz und knapp", "keep it short", "just answer",
		"no thinking", "don't think", "dont think",
	} {
		if strings.Contains(lower, kw) {
			return false
		}
	}

	for _, kw := range []string{
		"gedicht", "poem", "witz", "joke", "geschichte", "story",
		"reim", "limerick", "haiku",
	} {
		if strings.Contains(lower, kw) {
			return false
		}
	}

	for _, kw := range []string{
		"implement", "implementier", "plan", "analys", "architekt",
		"debug", "fix", "refactor", "design", "schritt", "umsetz",
		"how do i", "how would", "how to", "why does", "why is",
		"wie würde", "wie wuerde", "wie kann", "warum ",
		"explain", "erklär", "erklae",
	} {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	words := strings.Fields(goal)
	if len(words) >= 20 {
		return true
	}
	if len(goal) >= 120 {
		return true
	}

	return false
}
