package memory

import "strings"

// Kind categorizes a memory entry.
type Kind string

const (
	KindError       Kind = "error"
	KindPattern     Kind = "pattern"
	KindPreference  Kind = "preference"
	KindPersona     Kind = "persona"
	KindWorkflow    Kind = "workflow"
	KindInsight     Kind = "insight"
	KindAntiPattern Kind = "anti_pattern"
)

var validKinds = map[Kind]struct{}{
	KindError:       {},
	KindPattern:     {},
	KindPreference:  {},
	KindPersona:     {},
	KindWorkflow:    {},
	KindInsight:     {},
	KindAntiPattern: {},
}

// ParseKind normalizes and validates a kind string.
func ParseKind(raw string) (Kind, error) {
	k := Kind(strings.TrimSpace(strings.ToLower(raw)))
	if k == "" {
		return "", errKindRequired
	}
	if _, ok := validKinds[k]; !ok {
		return "", errUnknownKind
	}
	return k, nil
}

// MatchesKind reports whether filter is empty or matches entry kind.
func MatchesKind(filter string, kind Kind) bool {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return true
	}
	return string(kind) == filter
}
