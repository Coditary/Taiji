package memory

import (
	"strings"
	"unicode/utf8"
)

// resolveVerifiedFix returns the stored answer/value for a memory entry.
// Catalog kinds with requires_verified_fix=false may default to recall_text.
func resolveVerifiedFix(catalog *KindCatalog, kind Kind, recallText, verifiedFix string) (string, error) {
	verifiedFix = strings.TrimSpace(verifiedFix)
	if verifiedFix != "" {
		return verifiedFix, nil
	}
	if catalog != nil && catalog.RequiresVerifiedFix(kind) {
		return "", errFixRequired
	}
	if catalog == nil {
		switch kind {
		case KindPreference, KindInsight, KindPersona:
			recallText = strings.TrimSpace(recallText)
			if recallText == "" {
				return "", errFixRequired
			}
			return recallText, nil
		default:
			return "", errFixRequired
		}
	}
	recallText = strings.TrimSpace(recallText)
	if recallText == "" {
		return "", errFixRequired
	}
	return recallText, nil
}

// deriveMemoryTitle builds a short title when the model omits one.
func deriveMemoryTitle(kind Kind, recallText string) string {
	recallText = strings.TrimSpace(recallText)
	if recallText == "" {
		return string(kind)
	}
	line := recallText
	if idx := strings.IndexAny(line, "\n\r"); idx >= 0 {
		line = line[:idx]
	}
	line = strings.TrimSpace(line)
	const maxRunes = 72
	if utf8.RuneCountInString(line) <= maxRunes {
		return line
	}
	runes := []rune(line)
	return strings.TrimSpace(string(runes[:maxRunes])) + "…"
}
