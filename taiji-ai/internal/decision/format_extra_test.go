package decision

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatTextScoreAndEmpty(t *testing.T) {
	if FormatText(nil) != "no answers" {
		t.Fatal("nil response")
	}
	raw := json.RawMessage(`{
		"answers": {
			"s": {"type": "score", "score": 0.75, "probabilities": {"low": 0.1, "high": 0.9}}
		}
	}`)
	resp, err := ParseResponse(raw, "d", "http")
	if err != nil {
		t.Fatal(err)
	}
	resp.Backend = "http"
	resp.Driver = "laya"
	resp.Model = "m"
	text := FormatText(resp)
	for _, needle := range []string{"backend: http", "driver: laya", "model: m", "score: 0.750", "high=0.900"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("missing %q in %q", needle, text)
		}
	}
}

func TestToJSONRoundTrip(t *testing.T) {
	s, err := ToJSON(nil)
	if err != nil || s != "{}" {
		t.Fatalf("nil: %q err=%v", s, err)
	}
	resp := &Response{Model: "m", Answers: map[string]Answer{
		"a": {ID: "a", Type: QuestionNoul, Noul: &NoulAnswer{Value: 0.5}},
	}}
	out, err := ToJSON(resp)
	if err != nil || !strings.Contains(out, `"noul"`) {
		t.Fatalf("out=%q err=%v", out, err)
	}
}
