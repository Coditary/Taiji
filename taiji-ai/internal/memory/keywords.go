package memory

import (
	"strings"
	"unicode"
)

var defaultNameRecallQueries = []string{
	"wie heiße ich",
	"wie ist mein name",
	"wer bin ich",
	"what is my name",
	"kennst du meinen namen",
}

var defaultPersonaRecallQueries = []string{
	"wie heißt du",
	"wie ist dein name",
	"what is your name",
	"what are you called",
	"wer bist du",
}

// enrichMemoryEntry merges caller-provided metadata with auto-generated recall hints.
func enrichMemoryEntry(catalog *KindCatalog, kind Kind, title, recallText, verifiedFix string, tags, keywords, recallQueries []string) ([]string, []string, []string) {
	tags = dedupeTerms(tags)
	keywords = dedupeTerms(keywords)
	recallQueries = dedupePhrases(recallQueries)

	keywords = append(keywords, extractKeywordCandidates(title, recallText, verifiedFix)...)
	keywords = dedupeTerms(keywords)

	enrichName := catalog != nil && catalog.EnrichName(kind)
	if !enrichName && catalog == nil {
		enrichName = kind == KindPreference || kind == KindPersona
	}
	if enrichName && isNameLikeEntry(kind, title, recallText, verifiedFix, keywords) {
		keywords = append(keywords, "name", "vorname", "heißt", "identität")
		switch kind {
		case KindPersona:
			recallQueries = append(recallQueries, defaultPersonaRecallQueries...)
			tags = append(tags, "name", "persona", "assistant")
		default:
			recallQueries = append(recallQueries, defaultNameRecallQueries...)
			tags = append(tags, "name", "preference", "identity")
		}
	}

	keywords = ExpandTerms(keywords, defaultSynonymGroups)
	keywords = dedupeTerms(keywords)
	recallQueries = dedupePhrases(recallQueries)
	tags = dedupeTerms(tags)
	return tags, keywords, recallQueries
}

func isNameLikeEntry(kind Kind, title, recallText, verifiedFix string, keywords []string) bool {
	haystack := strings.ToLower(strings.Join([]string{title, recallText, verifiedFix, strings.Join(keywords, " ")}, " "))
	hints := []string{"name", "heiß", "vorname", "identität", "identity"}
	if kind == KindPersona {
		hints = append(hints, "assistant", "bot", "ki", "du heißt", "du bist", "persona")
	} else {
		hints = append(hints, "benutzer", "user")
	}
	for _, hint := range hints {
		if strings.Contains(haystack, hint) {
			return true
		}
	}
	return false
}

func extractKeywordCandidates(parts ...string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, part := range parts {
		for _, tok := range tokenize(part) {
			if len(tok) < 3 {
				continue
			}
			if _, ok := seen[tok]; ok {
				continue
			}
			seen[tok] = struct{}{}
			out = append(out, tok)
		}
	}
	return out
}

func dedupeTerms(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func dedupePhrases(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizePhrase(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizePhrase(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return ""
	}
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r)
	})
	return strings.Join(fields, " ")
}
