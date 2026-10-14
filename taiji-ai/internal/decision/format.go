package decision

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FormatText renders a decision response for tool output.
func FormatText(resp *Response) string {
	if resp == nil || len(resp.Answers) == 0 {
		return "no answers"
	}
	var b strings.Builder
	if resp.Backend != "" {
		fmt.Fprintf(&b, "backend: %s\n", resp.Backend)
	}
	if resp.Driver != "" {
		fmt.Fprintf(&b, "driver: %s\n", resp.Driver)
	}
	if resp.Model != "" {
		fmt.Fprintf(&b, "model: %s\n", resp.Model)
	}
	ids := make([]string, 0, len(resp.Answers))
	for id := range resp.Answers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ans := resp.Answers[id]
		fmt.Fprintf(&b, "\n[%s] type=%s\n", id, ans.Type)
		switch ans.Type {
		case QuestionChoice:
			if ans.Choice == nil {
				continue
			}
			fmt.Fprintf(&b, "choice: %s\n", ans.Choice.Choice)
			if ans.Choice.Confidence > 0 {
				fmt.Fprintf(&b, "confidence: %.3f\n", ans.Choice.Confidence)
			}
			if len(ans.Choice.Probabilities) > 0 {
				fmt.Fprintf(&b, "probabilities: %s\n", formatProbs(ans.Choice.Probabilities))
			}
		case QuestionNoul:
			if ans.Noul == nil {
				continue
			}
			fmt.Fprintf(&b, "noul: %.3f\n", ans.Noul.Value)
		case QuestionScore:
			if ans.Score == nil {
				continue
			}
			fmt.Fprintf(&b, "score: %.3f\n", ans.Score.Score)
			if len(ans.Score.Probabilities) > 0 {
				fmt.Fprintf(&b, "probabilities: %s\n", formatProbs(ans.Score.Probabilities))
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func formatProbs(probs map[string]float64) string {
	keys := make([]string, 0, len(probs))
	for k := range probs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%.3f", k, probs[k]))
	}
	return strings.Join(pairs, ", ")
}

// ToJSON returns a compact JSON view of the response.
func ToJSON(resp *Response) (string, error) {
	if resp == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
