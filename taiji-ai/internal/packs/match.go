package packs

import (
	"strings"
)

// KeywordExactHits returns nodes whose keywords appear as substrings in haystack (case-insensitive).
func KeywordExactHits(nodes []PackNode, haystack string) []PackNode {
	haystack = strings.ToLower(haystack)
	if haystack == "" {
		return nil
	}
	var hits []PackNode
	seen := map[string]bool{}
	for _, n := range nodes {
		if n.EffectiveInjection() != ModeKeyword {
			continue
		}
		terms := keywordTerms(n)
		for _, term := range terms {
			term = strings.ToLower(strings.TrimSpace(term))
			if len(term) < 2 {
				continue
			}
			if strings.Contains(haystack, term) {
				id := n.NodeID()
				if !seen[id] {
					seen[id] = true
					hits = append(hits, n)
				}
				break
			}
		}
	}
	return hits
}

func keywordTerms(n PackNode) []string {
	var out []string
	out = append(out, n.Meta.Keywords...)
	if name := strings.TrimSpace(n.Meta.Name); name != "" {
		out = append(out, name)
	}
	if when := strings.TrimSpace(n.Meta.When); when != "" {
		out = append(out, when)
	}
	return out
}
