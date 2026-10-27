package packs

import (
	"strings"
	"unicode"
)

func tokenize(text string) []string {
	text = strings.ToLower(text)
	var terms []string
	for _, field := range strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		field = strings.TrimSpace(field)
		if len(field) < 2 {
			continue
		}
		terms = append(terms, field)
	}
	return terms
}

func termFrequencies(text string) map[string]int {
	out := map[string]int{}
	for _, t := range tokenize(text) {
		out[t]++
	}
	return out
}
