package provider

import (
	"encoding/json"
	"testing"

	wujidriver "github.com/coditary/wuji-core/pkg/driver"
)

func TestWujiPayloadRoundTripTools(t *testing.T) {
	payload := map[string]any{
		"driver_id": "llama",
		"request": map[string]any{
			"messages": []Message{
				{Role: RoleUser, Content: "list files"},
			},
			"tools": []ToolDef{{
				Name:        "list_dir",
				Description: "list",
				Parameters:  map[string]any{"type": "object"},
			}},
			"tool_choice":    "required",
			"context_window": 32768,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		DriverID string               `json:"driver_id"`
		Request  wujidriver.TextRequest `json:"request"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Request.Tools) != 1 {
		t.Fatalf("tools = %#v", body.Request.Tools)
	}
	if body.Request.ToolChoice != "required" {
		t.Fatalf("tool_choice = %q", body.Request.ToolChoice)
	}
}
