package question

import (
	"fmt"
	"strings"
)

// ParseQuestions converts tool args into validated question infos.
func ParseQuestions(args map[string]any) ([]Info, error) {
	raw, ok := args["questions"]
	if !ok || raw == nil {
		return nil, ErrInvalidQuestions
	}
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, ErrInvalidQuestions
	}

	out := make([]Info, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("questions[%d]: expected object", i)
		}
		q, err := parseQuestion(i, m)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func parseQuestion(index int, m map[string]any) (Info, error) {
	q := Info{
		ID:       strings.TrimSpace(stringField(m, "id")),
		Header:   strings.TrimSpace(stringField(m, "header")),
		Question: strings.TrimSpace(stringField(m, "question")),
		Multiple: boolField(m, "multiple"),
	}
	if custom, ok := m["custom"]; ok {
		if b, ok := custom.(bool); ok {
			q.Custom = &b
		}
	}
	if q.Question == "" {
		return Info{}, fmt.Errorf("questions[%d]: question is required", index)
	}
	if q.Header == "" {
		q.Header = truncate(q.Question, 30)
	}

	rawOpts, ok := m["options"]
	if !ok || rawOpts == nil {
		return Info{}, fmt.Errorf("questions[%d]: options are required", index)
	}
	optItems, ok := rawOpts.([]any)
	if !ok || len(optItems) < 2 {
		return Info{}, fmt.Errorf("questions[%d]: at least two options are required", index)
	}

	opts := make([]Option, 0, len(optItems))
	for j, item := range optItems {
		om, ok := item.(map[string]any)
		if !ok {
			return Info{}, fmt.Errorf("questions[%d].options[%d]: expected object", index, j)
		}
		label := strings.TrimSpace(stringField(om, "label"))
		if label == "" {
			return Info{}, fmt.Errorf("questions[%d].options[%d]: label is required", index, j)
		}
		opts = append(opts, Option{
			ID:          strings.TrimSpace(stringField(om, "id")),
			Label:       label,
			Description: strings.TrimSpace(stringField(om, "description")),
		})
	}
	q.Options = opts
	return q, nil
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func boolField(m map[string]any, key string) bool {
	if v, ok := m[key]; ok && v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
