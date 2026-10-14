package provider

import (
	"encoding/json"
	"testing"

	wujidriver "github.com/coditary/wuji-core/pkg/driver"
)

func TestGenerateImagePayloadShape(t *testing.T) {
	payload := map[string]any{
		"driver_id": "",
		"request": wujidriver.ImageRequest{
			Task:   wujidriver.ImageTaskGenerate,
			Prompt: "a fox",
			Width:  512,
			Height: 512,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		DriverID string                  `json:"driver_id"`
		Request  wujidriver.ImageRequest `json:"request"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.Request.Prompt != "a fox" {
		t.Fatalf("prompt = %q", body.Request.Prompt)
	}
	if body.Request.Task != wujidriver.ImageTaskGenerate {
		t.Fatalf("task = %q", body.Request.Task)
	}
}
