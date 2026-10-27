package mcp

import (
	"encoding/json"
	"testing"
)

func TestFormatMCPContentsTextAndBlob(t *testing.T) {
	got := formatMCPContents([]any{
		map[string]any{"text": "line-a"},
		map[string]any{"blob": "base64-ish"},
		map[string]any{"other": true},
	})
	if got != "line-a\nbase64-ish" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatMCPContentToolResult(t *testing.T) {
	got := formatMCPContent([]any{
		map[string]any{"text": "pong"},
		map[string]any{"type": "image"},
	})
	if got != "pong" {
		t.Fatalf("got %q", got)
	}
}

func TestMessageIDMatches(t *testing.T) {
	tests := []struct {
		raw  any
		want int64
		ok   bool
	}{
		{float64(3), 3, true},
		{int(5), 5, true},
		{int64(7), 7, true},
		{json.Number("9"), 9, true},
		{"bad", 1, false},
		{float64(2), 3, false},
	}
	for _, tc := range tests {
		got := messageIDMatches(tc.raw, tc.want)
		if tc.ok && !got {
			t.Fatalf("raw=%v want match", tc.raw)
		}
		if !tc.ok && got {
			t.Fatalf("raw=%v should not match", tc.raw)
		}
	}
}
