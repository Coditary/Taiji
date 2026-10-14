package skills

import (
	"fmt"
	"strings"
)

// Match is one ranked skill candidate for a user message.
type Match struct {
	Name        string
	Description string
	Score       float32
	MatchSource string
	Fit         string
}

// FormatMatches renders ranked skill matches for tool observations or prompts.
func FormatMatches(matches []Match) string {
	if len(matches) == 0 {
		return "no skills matched above threshold"
	}
	lines := make([]string, 0, len(matches))
	for i, m := range matches {
		line := fmt.Sprintf("%d. %s — fit: %s (%.2f, %s)", i+1, m.Name, m.Fit, m.Score, m.MatchSource)
		if desc := strings.TrimSpace(m.Description); desc != "" {
			line += " — " + desc
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// assessSkillFit maps a score to a human-readable relevance tier.
func assessSkillFit(score, minScore float32) string {
	switch {
	case score >= 0.80:
		return "high"
	case score >= 0.65:
		return "medium"
	case score >= minScore:
		return "low"
	default:
		return "none"
	}
}
