package skills

import (
	"math"
	"sort"
	"strings"
)

const (
	skillBM25K1          = 1.2
	skillBM25B           = 0.75
	skillBM25MinRelative = 0.2
	skillBM25ScoreScale  = 0.88
)

type skillBM25Doc struct {
	skill  Skill
	terms  map[string]int
	length int
}

// searchSkillsBM25 ranks invokable skills over name, description, and body.
func searchSkillsBM25(skills []Skill, query string) []Match {
	query = strings.TrimSpace(query)
	if query == "" || len(skills) == 0 {
		return nil
	}
	queryTerms := skillTokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	var docs []skillBM25Doc
	for _, s := range skills {
		if s.DisableModelInvocation {
			continue
		}
		text := skillDocument(s)
		terms := skillTermFrequencies(text)
		if len(terms) == 0 {
			continue
		}
		length := 0
		for _, tf := range terms {
			length += tf
		}
		docs = append(docs, skillBM25Doc{skill: s, terms: terms, length: length})
	}
	if len(docs) == 0 {
		return nil
	}

	df := map[string]int{}
	for _, q := range queryTerms {
		count := 0
		for _, doc := range docs {
			if skillDocTermFreq(doc.terms, q) > 0 {
				count++
			}
		}
		if count > 0 {
			df[q] = count
		}
	}

	totalLen := 0
	for _, doc := range docs {
		totalLen += doc.length
	}
	avgDL := float64(totalLen) / float64(len(docs))
	nDocs := float64(len(docs))

	type scored struct {
		skill Skill
		raw   float64
	}
	var ranked []scored
	maxRaw := 0.0
	for _, doc := range docs {
		score := 0.0
		for _, q := range queryTerms {
			tf := skillDocTermFreq(doc.terms, q)
			if tf == 0 {
				continue
			}
			docFreq := float64(df[q])
			idf := math.Log((nDocs-docFreq+0.5)/(docFreq+0.5) + 1)
			denom := float64(tf) + skillBM25K1*(1-skillBM25B+skillBM25B*float64(doc.length)/avgDL)
			score += idf * (float64(tf) * (skillBM25K1 + 1)) / denom
		}
		if score <= 0 {
			continue
		}
		if score > maxRaw {
			maxRaw = score
		}
		ranked = append(ranked, scored{skill: doc.skill, raw: score})
	}
	if len(ranked) == 0 || maxRaw <= 0 {
		return nil
	}

	minRaw := maxRaw * skillBM25MinRelative
	out := make([]Match, 0, len(ranked))
	for _, hit := range ranked {
		if hit.raw < minRaw {
			continue
		}
		out = append(out, Match{
			Name:        hit.skill.Name,
			Description: hit.skill.Description,
			Score:       float32(hit.raw / maxRaw * skillBM25ScoreScale),
			MatchSource: "bm25",
			Fit:         assessSkillFit(float32(hit.raw/maxRaw*skillBM25ScoreScale), 0.55),
		})
	}
	sortSkillMatches(out)
	return out
}

func skillDocument(s Skill) string {
	name := strings.TrimSpace(s.Name)
	desc := strings.TrimSpace(s.Description)
	body := strings.TrimSpace(s.Body)
	parts := []string{name, name, desc, desc, body}
	for _, part := range strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	}) {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part, part)
		}
	}
	return strings.Join(parts, " ")
}

func sortSkillMatches(matches []Match) {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Name < matches[j].Name
		}
		return matches[i].Score > matches[j].Score
	})
}

func skillDocTermFreq(terms map[string]int, term string) int {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return 0
	}
	if tf, ok := terms[term]; ok {
		return tf
	}
	return 0
}
