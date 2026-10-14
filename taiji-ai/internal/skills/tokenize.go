package skills

import (
	"strings"
	"unicode"
)

var skillStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {}, "for": {},
	"from": {}, "how": {}, "i": {}, "in": {}, "is": {}, "it": {}, "me": {}, "my": {}, "of": {},
	"on": {}, "or": {}, "that": {}, "the": {}, "this": {}, "to": {}, "was": {}, "what": {}, "when": {},
	"where": {}, "who": {}, "why": {}, "with": {}, "you": {}, "your": {},
	"als": {}, "am": {}, "auch": {}, "auf": {}, "aus": {}, "bei": {}, "bin": {}, "bist": {},
	"das": {}, "dein": {}, "deine": {}, "dem": {}, "den": {}, "der": {}, "des": {}, "die": {}, "du": {},
	"ein": {}, "eine": {}, "einem": {}, "einen": {}, "einer": {}, "eines": {}, "er": {}, "es": {}, "für": {},
	"hat": {}, "hast": {}, "ich": {}, "im": {}, "ist": {}, "mir": {}, "mit": {}, "nach": {},
	"nicht": {}, "noch": {}, "sie": {}, "sind": {}, "und": {}, "uns": {}, "vom": {}, "von": {}, "war": {},
	"wenn": {}, "wie": {}, "wir": {}, "wo": {}, "zu": {}, "zum": {}, "zur": {},
}

func skillTokenize(text string) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return nil
	}
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	seen := map[string]struct{}{}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len(field) < 2 {
			continue
		}
		if _, stop := skillStopWords[field]; stop {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}

func skillTermFrequencies(text string) map[string]int {
	tokens := skillTokenize(text)
	if len(tokens) == 0 {
		return nil
	}
	out := map[string]int{}
	for _, tok := range tokens {
		out[tok]++
	}
	return out
}
