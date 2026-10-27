package session

import "strings"

// ParseTemplateCommand detects /template or /actor slash commands.
// Returns template id and true when the whole message is a template switch.
func ParseTemplateCommand(goal string) (templateID string, ok bool) {
	goal = strings.TrimSpace(goal)
	lower := strings.ToLower(goal)
	for _, prefix := range []string{"/template ", "/actor ", "/template\t", "/actor\t"} {
		if strings.HasPrefix(lower, prefix) {
			id := strings.TrimSpace(goal[len(prefix):])
			if id == "" {
				return "", false
			}
			return id, true
		}
	}
	if lower == "/template" || lower == "/actor" {
		return "", true
	}
	return "", false
}
