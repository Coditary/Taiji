package decision

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseQuestionsValidation(t *testing.T) {
	if _, err := ParseQuestions(nil); err == nil {
		t.Fatal("nil questions")
	}
	if _, err := ParseQuestions(42); err == nil {
		t.Fatal("wrong type")
	}
	if _, err := ParseQuestions(map[string]any{}); err == nil {
		t.Fatal("empty map")
	}
	if _, err := ParseQuestions([]any{}); err == nil {
		t.Fatal("empty slice")
	}
	if _, err := ParseQuestions(map[string]any{"x": "not-object"}); err == nil {
		t.Fatal("bad question object")
	}
}

func TestParseQuestionsTrimsCriteria(t *testing.T) {
	qs, err := ParseQuestions(map[string]any{
		"c": map[string]any{
			"type":         "choice",
			"instructions": "?",
			"criteria":     []any{"  billing  ", " tech "},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(qs[0].Criteria) != 2 || qs[0].Criteria[0] != "billing" || qs[0].Criteria[1] != "tech" {
		t.Fatalf("criteria=%v", qs[0].Criteria)
	}
}

func TestParseResponseEmptyAnswers(t *testing.T) {
	_, err := ParseResponse(json.RawMessage(`{"answers":{}}`), "d", "http")
	if err == nil {
		t.Fatal("expected empty answers error")
	}
}

func TestFormatTextChoiceAnswer(t *testing.T) {
	raw := json.RawMessage(`{
		"answers": {
			"route": {"type": "choice", "choice": "a", "probabilities": {"a": 0.8, "b": 0.2}}
		}
	}`)
	resp, err := ParseResponse(raw, "laya", "http")
	if err != nil {
		t.Fatal(err)
	}
	text := FormatText(resp)
	if !strings.Contains(text, "choice: a") {
		t.Fatalf("text=%q", text)
	}
}
