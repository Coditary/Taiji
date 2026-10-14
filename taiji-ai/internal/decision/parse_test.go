package decision

import (
	"encoding/json"
	"testing"
)

func TestParseQuestionsMap(t *testing.T) {
	questions, err := ParseQuestions(map[string]any{
		"route": map[string]any{
			"type":         "choice",
			"instructions": "Which team?",
			"criteria": map[string]any{
				"billing":   "Payment issues",
				"technical": "Bugs",
			},
		},
		"urgent": map[string]any{
			"type":         "noul",
			"instructions": "Is this urgent?",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 {
		t.Fatalf("len = %d", len(questions))
	}
	if questions[0].ID != "route" || questions[0].Type != QuestionChoice {
		t.Fatalf("route = %#v", questions[0])
	}
	if len(questions[0].Criteria) != 2 {
		t.Fatalf("criteria = %#v", questions[0].Criteria)
	}
}

func TestParseQuestionsArray(t *testing.T) {
	questions, err := ParseQuestions([]any{
		map[string]any{
			"id":           "route",
			"type":         "choice",
			"instructions": "Which team?",
			"criteria":     []any{"billing", "technical"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if questions[0].ID != "route" {
		t.Fatalf("id = %q", questions[0].ID)
	}
}

func TestParseResponse(t *testing.T) {
	raw := json.RawMessage(`{
		"model": "laya-multilingual",
		"answers": {
			"route": {
				"type": "choice",
				"choice": "billing",
				"probabilities": {"billing": 0.89, "technical": 0.08},
				"confidence": 0.83
			},
			"urgent": {
				"type": "noul",
				"noul": 0.94
			}
		}
	}`)
	resp, err := ParseResponse(raw, "laya", "wuji")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Answers["route"].Choice.Choice != "billing" {
		t.Fatalf("choice = %#v", resp.Answers["route"].Choice)
	}
	if resp.Answers["urgent"].Noul.Value != 0.94 {
		t.Fatalf("noul = %v", resp.Answers["urgent"].Noul.Value)
	}
}

func TestFormatText(t *testing.T) {
	resp, err := ParseResponse(json.RawMessage(`{
		"model": "laya-multilingual",
		"answers": {
			"urgent": {"type": "noul", "noul": 0.94}
		}
	}`), "laya", "wuji")
	if err != nil {
		t.Fatal(err)
	}
	text := FormatText(resp)
	if text == "" {
		t.Fatal("empty format")
	}
}
