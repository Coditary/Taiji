package provider

import (
	"encoding/json"
	"testing"
)

func TestDecisionRequestShape(t *testing.T) {
	payload := map[string]any{
		"driver_id": "laya",
		"request": DecisionRequest{
			Text: "Help! My payouts failed.",
			Questions: []DecisionQuestion{
				{
					ID:           "route",
					Type:         "choice",
					Instructions: "Which team should handle this?",
					Criteria:     []string{"billing", "technical", "sales"},
				},
				{
					ID:           "urgent",
					Type:         "noul",
					Instructions: "Does this convey urgency?",
				},
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		DriverID string `json:"driver_id"`
		Request  struct {
			Text      string             `json:"text"`
			Questions []DecisionQuestion `json:"questions"`
		} `json:"request"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.DriverID != "laya" {
		t.Fatalf("driver_id = %q", body.DriverID)
	}
	if len(body.Request.Questions) != 2 {
		t.Fatalf("questions = %d", len(body.Request.Questions))
	}
}
