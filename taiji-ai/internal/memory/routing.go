package memory

import "strings"

// FailureKinds are recalled automatically after tool/shell failures.
var FailureKinds = []Kind{KindError, KindAntiPattern}

// SuggestKindsForGoal heuristically picks memory kinds for a user message.
// Returns nil when memory recall is unlikely to help.
func SuggestKindsForGoal(goal string) []Kind {
	catalog := LoadKindCatalog(nil)
	if catalog != nil {
		return catalog.SuggestRecallKinds(goal)
	}
	return legacySuggestKinds(strings.ToLower(strings.TrimSpace(goal)))
}

func suggestKindsForGoal(s *Service, goal string) []Kind {
	if s != nil && s.kinds != nil {
		if kinds := s.kinds.SuggestRecallKinds(goal); len(kinds) > 0 {
			return kinds
		}
	}
	return SuggestKindsForGoal(goal)
}

func suggestStoreKinds(s *Service, message string) []Kind {
	if s == nil || s.kinds == nil {
		return nil
	}
	return s.kinds.SuggestStoreKinds(message)
}

// RoutingAppendix adds turn-specific memory guidance from the user goal.
func RoutingAppendix(s *Service, goal string) string {
	if s == nil || !s.Enabled() {
		return ""
	}
	kinds := suggestKindsForGoal(s, goal)
	storeKinds := suggestStoreKinds(s, goal)
	if len(kinds) == 0 && len(storeKinds) == 0 {
		return strings.TrimSpace(`### Memory routing (this turn)
If the user asks about personal facts, preferences, or past project knowledge you do not see in chat: call memory_recall before answering.
Pick the kind from the memory catalog (usually preference for user name/likes, persona for assistant name/role, pattern for repo conventions).
Do not say you lack information until recall has been tried.`)
	}
	if len(kinds) == 0 {
		var storeNames []string
		for _, k := range storeKinds {
			storeNames = append(storeNames, string(k))
		}
		return strings.TrimSpace("### Memory routing (this turn)\n" +
			"The user may be stating facts to remember. Consider memory_remember with kind: " + strings.Join(storeNames, ", ") + ".\n" +
			"Include keywords and recall_queries when storing preferences or persona facts.")
	}
	var names []string
	for _, k := range kinds {
		names = append(names, string(k))
	}
	queryHint := recallQueryHint(goal, kinds)
	return strings.TrimSpace("### Memory routing (this turn)\n" +
		"Call memory_recall now with kind filter: " + strings.Join(names, ", ") + ".\n" +
		queryHint +
		"Do not answer from chat context alone — check memory first.\n" +
		"Do not use error/anti_pattern memory for unrelated questions (e.g. preferences).\n" +
		"For assistant name questions use kind persona (NOT preference). For user name use kind preference.")
}

func recallQueryHint(goal string, kinds []Kind) string {
	intent := AnalyzeUserMessage(nil, goal)
	if intent != nil && intent.RecallQuery != "" {
		return "Suggested query: \"" + intent.RecallQuery + "\".\n"
	}
	return "Use the user's message as query.\n"
}
