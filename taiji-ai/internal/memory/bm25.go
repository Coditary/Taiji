package memory

import (
	"math"
	"strings"
	"unicode"
)

const (
	bm25K1          = 1.2
	bm25B           = 0.75
	bm25MinRelative = 0.2
	bm25ScoreScale  = 0.88
)

var bm25StopWords = map[string]struct{}{
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

type bm25Doc struct {
	entry  Entry
	terms  map[string]int
	length int
}

// searchBM25 ranks memory entries with Okapi BM25 over title, recall text, tags, keywords, recall queries, and verified fix.
func searchBM25(entries []Entry, query, kind string, synonymGroups [][]string) []Entry {
	query = strings.TrimSpace(query)
	if query == "" || len(entries) == 0 {
		return nil
	}
	queryTerms := ExpandTerms(tokenize(query), synonymGroups)
	if len(queryTerms) == 0 {
		return nil
	}

	var docs []bm25Doc
	for _, e := range entries {
		if !MatchesKind(kind, e.Kind) {
			continue
		}
		text := entryDocument(e)
		terms := termFrequencies(text)
		if len(terms) == 0 {
			continue
		}
		length := 0
		for _, tf := range terms {
			length += tf
		}
		docs = append(docs, bm25Doc{entry: e, terms: terms, length: length})
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
		entry Entry
		raw   float64
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
			denom := float64(tf) + bm25K1*(1-bm25B+bm25B*float64(doc.length)/avgDL)
			score += idf * (float64(tf) * (bm25K1 + 1)) / denom
		}
		if score <= 0 {
			continue
		}
		if score > maxRaw {
			maxRaw = score
		}
		ranked = append(ranked, scored{entry: doc.entry, raw: score})
	}
	if len(ranked) == 0 || maxRaw <= 0 {
		return nil
	}

	minRaw := maxRaw * bm25MinRelative
	out := make([]Entry, 0, len(ranked))
	for _, hit := range ranked {
		if hit.raw < minRaw {
			continue
		}
		copy := hit.entry
		copy.Score = float32(hit.raw / maxRaw * bm25ScoreScale)
		copy.MatchSource = "bm25"
		out = append(out, copy)
	}
	sortEntries(out)
	return out
}

func entryDocument(e Entry) string {
	parts := []string{e.Title, e.RecallText, e.VerifiedFix}
	if len(e.Tags) > 0 {
		parts = append(parts, strings.Join(e.Tags, " "))
	}
	if len(e.Keywords) > 0 {
		// Weight curated keywords higher than incidental body text.
		kw := strings.Join(e.Keywords, " ")
		parts = append(parts, kw, kw)
	}
	if len(e.RecallQueries) > 0 {
		parts = append(parts, strings.Join(e.RecallQueries, " "))
	}
	return strings.Join(parts, " ")
}

func tokenize(text string) []string {
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
		if _, stop := bm25StopWords[field]; stop {
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

func termFrequencies(text string) map[string]int {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return nil
	}
	// Keep duplicate terms in the document so repeated concepts score higher.
	all := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := map[string]int{}
	for _, tok := range all {
		tok = strings.TrimSpace(tok)
		if len(tok) < 2 {
			continue
		}
		if _, stop := bm25StopWords[tok]; stop {
			continue
		}
		out[tok]++
	}
	return out
}

func docTermFreq(terms map[string]int, queryTerm string) int {
	if terms == nil {
		return 0
	}
	if tf, ok := terms[queryTerm]; ok {
		return tf
	}
	total := 0
	for docTerm, tf := range terms {
		if termMatches(queryTerm, docTerm) {
			total += tf
		}
	}
	return total
}

func termMatches(queryTerm, docTerm string) bool {
	queryTerm = strings.ToLower(strings.TrimSpace(queryTerm))
	docTerm = strings.ToLower(strings.TrimSpace(docTerm))
	if queryTerm == "" || docTerm == "" {
		return false
	}
	if queryTerm == docTerm {
		return true
	}
	if len([]rune(queryTerm)) >= 4 && len([]rune(docTerm)) >= 4 {
		return sharedPrefixRunes(queryTerm, docTerm, 4)
	}
	return false
}

func sharedPrefixRunes(a, b string, n int) bool {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) < n || len(br) < n {
		return false
	}
	for i := 0; i < n; i++ {
		if ar[i] != br[i] {
			return false
		}
	}
	return true
}
