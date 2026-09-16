package rag

import "sort"

func mergeHits(hits []Hit, topK int) []Hit {
	if len(hits) == 0 {
		return nil
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].Score > hits[j].Score
	})
	seen := map[string]struct{}{}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		key := h.Scope + "\x00" + h.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, h)
		if topK > 0 && len(out) >= topK {
			break
		}
	}
	return out
}
