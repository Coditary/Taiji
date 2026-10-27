package session

import (
	"testing"

	"github.com/coditary/taiji/internal/events"
)

func TestMetricsFromDataCoercions(t *testing.T) {
	ev := events.Event{Data: map[string]any{
		"input_tokens":       float64(12),
		"output_tokens":      int64(3),
		"total_duration_ms":  100,
		"estimated_cost_usd": 0.01,
		"had_thinking":       true,
		"unknown":            "skip",
	}}
	m := metricsFromData(ev.Data)
	if m == nil || m.InputTokens != 12 || m.OutputTokens != 3 || m.TotalDurationMs != 100 {
		t.Fatalf("metrics=%+v", m)
	}
	if m.EstimatedCostUSD != 0.01 || !m.HadThinking {
		t.Fatalf("cost/thinking=%+v", m)
	}
}

func TestToInt64ToFloat64ToBool(t *testing.T) {
	if toInt64(5) != 5 || toInt64("x") != 0 {
		t.Fatal("toInt64")
	}
	if toFloat64(int(2)) != 2 || toFloat64("n") != 0 {
		t.Fatal("toFloat64")
	}
	if !toBool(true) || toBool(1) {
		t.Fatal("toBool")
	}
}
