package packs

import (
	"math"
	"sort"
	"strings"
)

const (
	packBM25K1          = 1.2
	packBM25B           = 0.75
	packBM25MinRelative = 0.2
	packBM25ScoreScale  = 0.88
)

// ScoredNode is a BM25-ranked pack node.
type ScoredNode struct {
	Node  PackNode
	Score float32
}

type packBM25Doc struct {
	node   PackNode
	terms  map[string]int
	length int
}

// SearchBM25 ranks nodes for keyword-mode fallback (name, description, when, keywords, body).
func SearchBM25(nodes []PackNode, query string, minScore float32) []ScoredNode {
	query = strings.TrimSpace(query)
	if query == "" || len(nodes) == 0 {
		return nil
	}
	queryTerms := tokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	var docs []packBM25Doc
	for _, n := range nodes {
		text := nodeDocument(n)
		terms := termFrequencies(text)
		if len(terms) == 0 {
			continue
		}
		length := 0
		for _, tf := range terms {
			length += tf
		}
		docs = append(docs, packBM25Doc{node: n, terms: terms, length: length})
	}
	if len(docs) == 0 {
		return nil
	}

	df := map[string]int{}
	for _, q := range queryTerms {
		count := 0
		for _, doc := range docs {
			if docTermFreq(doc.terms, q) > 0 {
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
		node PackNode
		raw  float64
	}
	var ranked []scored
	maxRaw := 0.0
	for _, doc := range docs {
		score := 0.0
		for _, q := range queryTerms {
			tf := docTermFreq(doc.terms, q)
			if tf == 0 {
				continue
			}
			docFreq := float64(df[q])
			idf := math.Log((nDocs-docFreq+0.5)/(docFreq+0.5) + 1)
			denom := float64(tf) + packBM25K1*(1-packBM25B+packBM25B*float64(doc.length)/avgDL)
			score += idf * (float64(tf) * (packBM25K1 + 1)) / denom
		}
		if score <= 0 {
			continue
		}
		if score > maxRaw {
			maxRaw = score
		}
		ranked = append(ranked, scored{node: doc.node, raw: score})
	}
	if len(ranked) == 0 || maxRaw <= 0 {
		return nil
	}

	minRaw := maxRaw * packBM25MinRelative
	out := make([]ScoredNode, 0, len(ranked))
	for _, hit := range ranked {
		if hit.raw < minRaw {
			continue
		}
		norm := float32(hit.raw / maxRaw * packBM25ScoreScale)
		if minScore > 0 && norm < minScore {
			continue
		}
		out = append(out, ScoredNode{Node: hit.node, Score: norm})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Node.NodeID() < out[j].Node.NodeID()
		}
		return out[i].Score > out[j].Score
	})
	return out
}

func nodeDocument(n PackNode) string {
	name := strings.TrimSpace(n.Meta.Name)
	desc := strings.TrimSpace(n.Meta.Description)
	when := strings.TrimSpace(n.Meta.When)
	body := strings.TrimSpace(n.Body)
	parts := []string{name, name, desc, desc, when, when}
	for _, kw := range n.Meta.Keywords {
		kw = strings.TrimSpace(kw)
		if kw != "" {
			parts = append(parts, kw, kw)
		}
	}
	parts = append(parts, body)
	return strings.Join(parts, " ")
}

func docTermFreq(terms map[string]int, term string) int {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return 0
	}
	if tf, ok := terms[term]; ok {
		return tf
	}
	return 0
}
