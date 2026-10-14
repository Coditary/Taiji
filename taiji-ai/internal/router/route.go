package router

import "strings"

// RouteTurn handles only explicit control commands. Natural-language intent
// (tools, memory, skills, RAG, etc.) is left to the agent loop and LLM so
// routing works across languages without maintained keyword lists.
func RouteTurn(goal string) Plan {
	goal = strings.TrimSpace(goal)
	lower := strings.ToLower(goal)
	if goal == "" {
		return Plan{Action: ActionUnknown, Reason: "empty user message"}
	}

	if isStopCommand(lower) {
		return Plan{Action: ActionStop, Reason: "user requested stop", Message: "Gestoppt."}
	}

	if expectsHook(lower) {
		return Plan{Action: ActionHook, Reason: "explicit hook command"}
	}

	return Plan{Action: ActionUnknown, Reason: "defer to agent — interpret intent and choose tools"}
}

func isStopCommand(lower string) bool {
	// Exact control tokens only — not natural-language routing.
	for _, phrase := range []string{"stop", "cancel", "abort", "quit", "exit"} {
		if lower == phrase {
			return true
		}
	}
	if strings.HasPrefix(lower, "/stop") {
		return true
	}
	return false
}

func expectsHook(lower string) bool {
	return strings.HasPrefix(lower, "/hook ") || strings.HasPrefix(lower, "!hook ")
}
