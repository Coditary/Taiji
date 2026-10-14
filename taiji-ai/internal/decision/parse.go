package decision

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ParseQuestions accepts laya-style maps or wuji-style arrays.
func ParseQuestions(raw any) ([]Question, error) {
	if raw == nil {
		return nil, fmt.Errorf("questions is required")
	}
	switch v := raw.(type) {
	case map[string]any:
		return parseQuestionMap(v)
	case []any:
		return parseQuestionSlice(v)
	default:
		return nil, fmt.Errorf("questions must be an object or array")
	}
}

func parseQuestionMap(m map[string]any) ([]Question, error) {
	if len(m) == 0 {
		return nil, fmt.Errorf("questions is empty")
	}
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Question, 0, len(ids))
	for _, id := range ids {
		spec, ok := m[id].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("question %q must be an object", id)
		}
		q, err := parseQuestionSpec(id, spec)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func parseQuestionSlice(items []any) ([]Question, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("questions is empty")
	}
	out := make([]Question, 0, len(items))
	for i, item := range items {
		spec, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("questions[%d] must be an object", i)
		}
		id, _ := spec["id"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, fmt.Errorf("questions[%d].id is required", i)
		}
		q, err := parseQuestionSpec(id, spec)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func parseQuestionSpec(id string, spec map[string]any) (Question, error) {
	qType, _ := spec["type"].(string)
	qType = strings.TrimSpace(strings.ToLower(qType))
	if qType == "" {
		return Question{}, fmt.Errorf("question %q: type is required", id)
	}
	instructions, _ := spec["instructions"].(string)
	instructions = strings.TrimSpace(instructions)
	criteria, err := parseCriteria(spec["criteria"])
	if err != nil {
		return Question{}, fmt.Errorf("question %q: %w", id, err)
	}
	switch QuestionType(qType) {
	case QuestionChoice, QuestionNoul, QuestionScore:
	default:
		return Question{}, fmt.Errorf("question %q: unsupported type %q", id, qType)
	}
	if QuestionType(qType) == QuestionChoice && len(criteria) == 0 {
		return Question{}, fmt.Errorf("question %q: choice questions require criteria", id)
	}
	return Question{
		ID:           id,
		Type:         QuestionType(qType),
		Instructions: instructions,
		Criteria:     criteria,
	}, nil
}

func parseCriteria(raw any) ([]string, error) {
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case []string:
		return trimStrings(v), nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return nil, fmt.Errorf("criteria entries must be non-empty strings")
			}
			out = append(out, strings.TrimSpace(s))
		}
		return out, nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return keys, nil
	default:
		return nil, fmt.Errorf("criteria must be an array or object")
	}
}

func trimStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ParseResponse normalizes laya/wuji JSON into Response.
func ParseResponse(raw json.RawMessage, driver, backend string) (*Response, error) {
	var doc struct {
		Model   string                     `json:"model"`
		Answers map[string]json.RawMessage `json:"answers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("decode decision response: %w", err)
	}
	if len(doc.Answers) == 0 {
		return nil, fmt.Errorf("empty decision response")
	}
	answers := make(map[string]Answer, len(doc.Answers))
	for id, body := range doc.Answers {
		ans, err := parseAnswer(id, body)
		if err != nil {
			return nil, err
		}
		answers[id] = ans
	}
	return &Response{
		Model:   strings.TrimSpace(doc.Model),
		Driver:  driver,
		Backend: backend,
		Answers: answers,
	}, nil
}

func parseAnswer(id string, raw json.RawMessage) (Answer, error) {
	var doc struct {
		Type          string             `json:"type"`
		Choice        string             `json:"choice"`
		Probabilities map[string]float64 `json:"probabilities"`
		Confidence    float64            `json:"confidence"`
		Noul          float64            `json:"noul"`
		Score         float64            `json:"score"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Answer{}, fmt.Errorf("answer %q: %w", id, err)
	}
	qType := QuestionType(strings.TrimSpace(strings.ToLower(doc.Type)))
	ans := Answer{ID: id, Type: qType}
	switch qType {
	case QuestionChoice:
		ans.Choice = &ChoiceAnswer{
			Choice:        strings.TrimSpace(doc.Choice),
			Probabilities: doc.Probabilities,
			Confidence:    doc.Confidence,
		}
	case QuestionNoul:
		ans.Noul = &NoulAnswer{Value: doc.Noul}
	case QuestionScore:
		ans.Score = &ScoreAnswer{
			Score:         doc.Score,
			Probabilities: doc.Probabilities,
		}
	default:
		return Answer{}, fmt.Errorf("answer %q: unsupported type %q", id, doc.Type)
	}
	return ans, nil
}
