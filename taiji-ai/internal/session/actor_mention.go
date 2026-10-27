package session

import (
	"regexp"
	"strings"
)

var actorMentionRE = regexp.MustCompile(`@([a-zA-Z0-9][a-zA-Z0-9_-]*)`)

// ResolveActorMention finds the first @template-id in goal and returns the template id and goal with that token removed.
func ResolveActorMention(projectRoot, goal string) (templateID string, stripped string) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return "", goal
	}
	m := actorMentionRE.FindStringSubmatch(goal)
	if len(m) < 2 {
		return "", goal
	}
	candidate := strings.TrimSpace(m[1])
	if candidate == "" {
		return "", goal
	}
	for _, t := range ListChatTemplates(projectRoot) {
		if strings.EqualFold(strings.TrimSpace(t.ID), candidate) {
			stripped = strings.TrimSpace(actorMentionRE.ReplaceAllString(goal, ""))
			if stripped == "" {
				stripped = goal
			}
			return t.ID, stripped
		}
	}
	return "", goal
}

// MergeActorState copies keys from overlay into base (overlay wins on conflict).
func MergeActorState(base, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}
