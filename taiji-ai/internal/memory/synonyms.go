package memory

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type synonymFile struct {
	Groups [][]string `yaml:"groups"`
}

// defaultSynonymGroups bridge common query terms for memory recall.
var defaultSynonymGroups = [][]string{
	{"name", "vorname", "heißt", "heiße", "nennen", "identität", "identity", "benutzer", "user"},
	{"farbe", "lieblingsfarbe", "favorite", "favourite", "colour", "color"},
	{"sprache", "language", "deutsch", "english", "locale"},
	{"projekt", "project", "repo", "repository", "codebase"},
}

// LoadSynonymGroups merges built-in groups with an optional YAML file.
func LoadSynonymGroups(path string) [][]string {
	groups := cloneSynonymGroups(defaultSynonymGroups)
	path = strings.TrimSpace(path)
	if path == "" {
		return groups
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return groups
	}
	var file synonymFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return groups
	}
	for _, group := range file.Groups {
		if merged := normalizeSynonymGroup(group); len(merged) > 0 {
			groups = append(groups, merged)
		}
	}
	return groups
}

// ExpandTerms adds synonym-group members for every matching query term.
func ExpandTerms(terms []string, groups [][]string) []string {
	if len(terms) == 0 || len(groups) == 0 {
		return terms
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(terms)*2)
	add := func(term string) {
		term = strings.ToLower(strings.TrimSpace(term))
		if len(term) < 2 {
			return
		}
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}
	for _, term := range terms {
		add(term)
		for _, group := range groups {
			if !groupContains(group, term) {
				continue
			}
			for _, sibling := range group {
				add(sibling)
			}
		}
	}
	return out
}

func normalizeSynonymGroup(group []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(group))
	for _, term := range group {
		term = strings.ToLower(strings.TrimSpace(term))
		if len(term) < 2 {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}
	return out
}

func groupContains(group []string, term string) bool {
	term = strings.ToLower(strings.TrimSpace(term))
	for _, candidate := range group {
		if term == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
		if termMatches(term, candidate) {
			return true
		}
	}
	return false
}

func cloneSynonymGroups(groups [][]string) [][]string {
	out := make([][]string, len(groups))
	for i, group := range groups {
		out[i] = append([]string(nil), group...)
	}
	return out
}
