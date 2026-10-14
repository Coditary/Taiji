package memory

import (
	"regexp"
	"strings"
)

// IntentAction classifies what memory should do for a user message.
type IntentAction string

const (
	IntentRecall IntentAction = "recall"
	IntentStore  IntentAction = "store"
)

// UserIntent is a deterministic memory routing decision for one user message.
type UserIntent struct {
	Action      IntentAction
	Kinds       []Kind
	RecallQuery string
	Store       *RememberRequest
}

var (
	reUserNameStore = regexp.MustCompile(`(?i)(?:ich hei(?:ß|ss)e|my name is|merk dir(?:,)? (?:dass )?ich hei(?:ß|ss)e)\s+([^\s.!?]+)`)
	reFavoriteColor = regexp.MustCompile(`(?i)(?:meine lieblingsfarbe ist|my favorite colou?r is|ich mag(?: lieber)?)\s+(.+?)[\s.!?]*$`)
	reLanguagePref = regexp.MustCompile(`(?i)(?:ich spreche|my language is|bevorzug(?:e)? (?:die )?sprache)\s+(.+?)[\s.!?]*$`)
	reRepoPattern = regexp.MustCompile(`(?i)(?:in (?:this|dem|diesem) (?:repo|projekt)|here we use|bei uns(?: nutzen| verwenden| benutzen)? wir?)\s*(?:nutzen|verwenden|benutzen|use)?\s*(?:wir\s+)?(.+?)[\s.!?]*$`)
	reAssistantNameRecall = regexp.MustCompile(`(?i)(?:wie hei(?:ß|ss)t du|wie ist dein name|what(?:'s| is) your name|what are you called|wer bist du|dein name|your name)`)
	reUserNameRecall = regexp.MustCompile(`(?i)(?:wie hei(?:ß|ss)e ich|wie ist mein name|what(?:'s| is) my name|who am i|wer bin ich|kennst du mich|mein name)`)
	reColorRecall = regexp.MustCompile(`(?i)(?:lieblingsfarbe|favorite colou?r|was mag ich|what do i like)`)
	reLanguageRecall = regexp.MustCompile(`(?i)(?:welche sprache|which language|what language)`)
	rePatternRecall = regexp.MustCompile(`(?i)(?:in (?:this|dem|diesem) (?:repo|projekt)|repo convention|how do we do things here|was nutzen wir)`)
)

// AnalyzeUserMessage picks recall/store intents from a user message using catalog hints and extractors.
func AnalyzeUserMessage(catalog *KindCatalog, message string) *UserIntent {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	lower := strings.ToLower(message)

	if store := extractStoreIntent(catalog, message, lower); store != nil {
		return store
	}
	if recall := extractRecallIntent(catalog, message, lower); recall != nil {
		return recall
	}
	return nil
}

func extractStoreIntent(catalog *KindCatalog, message, lower string) *UserIntent {
	if looksLikeQuestion(lower) {
		return tryExplicitStoreOnly(message)
	}
	if store := tryExtractPersonaStore(message); store != nil {
		return store
	}
	if store := tryExtractUserNameStore(message); store != nil {
		return store
	}
	if store := tryExtractFavoriteColorStore(message); store != nil {
		return store
	}
	if store := tryExtractLanguageStore(message); store != nil {
		return store
	}
	if store := tryExtractPatternStore(message); store != nil {
		return store
	}

	kinds := suggestStoreKindsFromCatalog(catalog, lower)
	if len(kinds) == 0 {
		return nil
	}
	return &UserIntent{
		Action: IntentStore,
		Kinds:  kinds,
		Store: &RememberRequest{
			Kind:       string(kinds[0]),
			Title:      deriveMemoryTitle(kinds[0], message),
			RecallText: message,
			Scope:      "local",
		},
	}
}

func extractRecallIntent(catalog *KindCatalog, message, lower string) *UserIntent {
	switch {
	case reAssistantNameRecall.MatchString(lower):
		return &UserIntent{
			Action:      IntentRecall,
			Kinds:       []Kind{KindPersona},
			RecallQuery: "assistant name persona identity",
		}
	case reUserNameRecall.MatchString(lower):
		return &UserIntent{
			Action:      IntentRecall,
			Kinds:       []Kind{KindPreference},
			RecallQuery: "user name identity vorname",
		}
	case reColorRecall.MatchString(lower):
		return &UserIntent{
			Action:      IntentRecall,
			Kinds:       []Kind{KindPreference},
			RecallQuery: "favorite color lieblingsfarbe preference",
		}
	case reLanguageRecall.MatchString(lower):
		return &UserIntent{
			Action:      IntentRecall,
			Kinds:       []Kind{KindPreference},
			RecallQuery: "language sprache preference locale",
		}
	case rePatternRecall.MatchString(lower) && !looksLikeQuestion(lower) && !looksLikeRepoExploration(lower):
		return &UserIntent{
			Action:      IntentRecall,
			Kinds:       []Kind{KindPattern},
			RecallQuery: "repo convention pattern project",
		}
	}

	kinds := suggestRecallKindsFromCatalog(catalog, lower)
	if len(kinds) == 0 {
		return nil
	}
	return &UserIntent{
		Action:      IntentRecall,
		Kinds:       kinds,
		RecallQuery: recallQueryForKinds(message, kinds),
	}
}

func tryExtractPersonaStore(message string) *UserIntent {
	// Direct patterns like "Du heißt jetzt Luma"
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)^du hei(?:ß|ss)t(?:\s+(?:ab jetzt|nun|jetzt|now))?\s+(.+?)[\s.!?]*$`),
		regexp.MustCompile(`(?i)^du bist(?:\s+(?:ab jetzt|nun|jetzt|now))?\s+(.+?)[\s.!?]*$`),
		regexp.MustCompile(`(?i)^(?:from now on )?you are called\s+(.+?)[\s.!?]*$`),
		regexp.MustCompile(`(?i)^nenn dich(?:\s+(?:ab jetzt|nun|jetzt))?\s+(.+?)[\s.!?]*$`),
	} {
		m := re.FindStringSubmatch(strings.TrimSpace(message))
		if len(m) < 2 {
			continue
		}
		name := cleanStoredValue(m[1])
		if name == "" {
			continue
		}
		return personaStoreIntent(name)
	}
	return nil
}

func tryExtractUserNameStore(message string) *UserIntent {
	m := reUserNameStore.FindStringSubmatch(message)
	if len(m) < 2 {
		return nil
	}
	name := cleanStoredValue(m[1])
	if name == "" {
		return nil
	}
	return &UserIntent{
		Action: IntentStore,
		Kinds:  []Kind{KindPreference},
		Store: &RememberRequest{
			Kind:        string(KindPreference),
			Title:       "User name: " + name,
			RecallText:  "The user's name is " + name + ".",
			VerifiedFix: name,
			Scope:       "local",
		},
	}
}

func tryExtractFavoriteColorStore(message string) *UserIntent {
	m := reFavoriteColor.FindStringSubmatch(message)
	if len(m) < 2 {
		return nil
	}
	color := cleanStoredValue(m[1])
	if color == "" {
		return nil
	}
	return &UserIntent{
		Action: IntentStore,
		Kinds:  []Kind{KindPreference},
		Store: &RememberRequest{
			Kind:        string(KindPreference),
			Title:       "Favorite color: " + color,
			RecallText:  "The user's favorite color is " + color + ".",
			VerifiedFix: color,
			Scope:       "local",
		},
	}
}

func tryExtractLanguageStore(message string) *UserIntent {
	m := reLanguagePref.FindStringSubmatch(message)
	if len(m) < 2 {
		return nil
	}
	lang := cleanStoredValue(m[1])
	if lang == "" {
		return nil
	}
	return &UserIntent{
		Action: IntentStore,
		Kinds:  []Kind{KindPreference},
		Store: &RememberRequest{
			Kind:        string(KindPreference),
			Title:       "Preferred language: " + lang,
			RecallText:  "The user prefers language " + lang + ".",
			VerifiedFix: lang,
			Scope:       "local",
		},
	}
}

func tryExtractPatternStore(message string) *UserIntent {
	lower := strings.ToLower(strings.TrimSpace(message))
	if looksLikeQuestion(lower) || looksLikeRepoExploration(lower) {
		return nil
	}
	m := reRepoPattern.FindStringSubmatch(message)
	if len(m) < 2 {
		return nil
	}
	pattern := cleanStoredValue(m[1])
	if pattern == "" || isRepoExplorationFragment(pattern) {
		return nil
	}
	return &UserIntent{
		Action: IntentStore,
		Kinds:  []Kind{KindPattern},
		Store: &RememberRequest{
			Kind:        string(KindPattern),
			Title:       "Repo convention: " + pattern,
			RecallText:  "In this repo: " + pattern,
			VerifiedFix: pattern,
			Scope:       "local",
		},
	}
}

func personaStoreIntent(name string) *UserIntent {
	return &UserIntent{
		Action: IntentStore,
		Kinds:  []Kind{KindPersona},
		Store: &RememberRequest{
			Kind:        string(KindPersona),
			Title:       "Assistant name: " + name,
			RecallText:  "The assistant's name is " + name + ".",
			VerifiedFix: name,
			Scope:       "local",
		},
	}
}

func recallQueryForKinds(message string, kinds []Kind) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "memory recall"
	}
	for _, kind := range kinds {
		switch kind {
		case KindPersona:
			return "assistant name persona identity"
		case KindPreference:
			if reColorRecall.MatchString(strings.ToLower(message)) {
				return "favorite color lieblingsfarbe preference"
			}
			return "user name preference identity"
		case KindPattern:
			return "repo convention pattern project"
		case KindWorkflow:
			return "workflow steps how to"
		}
	}
	return message
}

func suggestStoreKindsFromCatalog(catalog *KindCatalog, lower string) []Kind {
	if catalog == nil {
		return nil
	}
	return catalog.SuggestStoreKinds(lower)
}

func suggestRecallKindsFromCatalog(catalog *KindCatalog, lower string) []Kind {
	if catalog == nil {
		return legacySuggestKinds(lower)
	}
	return catalog.SuggestRecallKinds(lower)
}

func looksLikeQuestion(lower string) bool {
	lower = strings.TrimSpace(lower)
	if strings.HasSuffix(lower, "?") {
		return true
	}
	for _, prefix := range []string{"wie ", "was ", "wer ", "what ", "who ", "which ", "welche ", "welcher ", "welches "} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	for _, phrase := range []string{
		"kannst du", "könntest du", "können sie", "can you", "could you", "would you",
		"sag mir", "zeig mir", "tell me", "show me", "list ", "liste ",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func looksLikeRepoExploration(lower string) bool {
	for _, phrase := range []string{
		"drin ist", "drin liegt", "was ist drin", "was liegt", "alles in dem repo",
		"alles in diesem repo", "alles im repo", "inhalt des repo", "repo drin",
		"what is in the repo", "what's in the repo", "contents of the repo",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func isRepoExplorationFragment(pattern string) bool {
	lower := strings.ToLower(strings.TrimSpace(pattern))
	for _, phrase := range []string{"drin ist", "drin liegt", "drin", "alles", "inhalt"} {
		if lower == phrase {
			return true
		}
	}
	return false
}

func tryExplicitStoreOnly(message string) *UserIntent {
	if store := tryExtractPersonaStore(message); store != nil {
		return store
	}
	if store := tryExtractUserNameStore(message); store != nil {
		return store
	}
	if store := tryExtractFavoriteColorStore(message); store != nil {
		return store
	}
	if store := tryExtractLanguageStore(message); store != nil {
		return store
	}
	return tryExtractPatternStore(message)
}

func cleanStoredValue(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"'.,!?`)
	raw = strings.TrimSpace(raw)
	if len(raw) > 120 {
		raw = raw[:120]
	}
	return raw
}
