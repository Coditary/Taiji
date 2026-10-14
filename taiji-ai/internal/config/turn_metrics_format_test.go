package config

import "testing"

func TestFormatTurnMetricsDefaultFields(t *testing.T) {
	text := FormatTurnMetrics(TurnMetricsValues{
		TotalDurationMs:      17700,
		WaitDurationMs:       9000,
		GenerationDurationMs: 1100,
		OutputTokens:         128,
		TokensPerSec:         38.5,
	}, DefaultUIConfig().TurnMetrics)
	want := "18s total · 9.0s loading · 1.1s generating · 128 tok\n38.5 tok/s"
	if text != want {
		t.Fatalf("text = %q want %q", text, want)
	}
}

func TestFormatTurnMetricsCustomFields(t *testing.T) {
	cfg := TurnMetricsUIConfig{Fields: []string{"total", "output_tokens"}}
	cfg.normalize()
	text := FormatTurnMetrics(TurnMetricsValues{
		TotalDurationMs: 900,
		OutputTokens:    42,
		TokensPerSec:    10,
	}, cfg)
	if text != "900ms total · 42 tok" {
		t.Fatalf("text = %q", text)
	}
}

func TestFormatTurnMetricsDisabled(t *testing.T) {
	enabled := false
	cfg := TurnMetricsUIConfig{Enabled: &enabled, Fields: []string{"total"}}
	text := FormatTurnMetrics(TurnMetricsValues{TotalDurationMs: 1000}, cfg)
	if text != "" {
		t.Fatalf("text = %q", text)
	}
}

func TestNormalizeTurnMetricFieldsAliases(t *testing.T) {
	got := normalizeTurnMetricFields([]string{"time", "tokens", "tps", "writing", "cost"})
	want := []string{
		TurnMetricFieldTotal,
		TurnMetricFieldOutputTokens,
		TurnMetricFieldTokensPerSec,
		TurnMetricFieldGeneration,
		TurnMetricFieldCost,
	}
	if len(got) != len(want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("field %d = %q want %q", i, got[i], want[i])
		}
	}
}

func TestFormatTurnMetricsPresetDev(t *testing.T) {
	cfg := TurnMetricsUIConfig{Preset: "dev"}
	cfg.normalize()
	lines := FormatTurnMetricsLines(TurnMetricsValues{
		TotalDurationMs:      5000,
		ThinkingDurationMs:   1000,
		GenerationDurationMs: 3000,
		OutputTokens:         80,
		InputTokens:          1200,
		TokensPerSec:         26.6,
		ToolCalls:            2,
		ToolDurationMs:       400,
		EstimatedCostUSD:     0.0042,
		Steps:                3,
		HadThinking:          true,
	}, cfg, 0)
	if len(lines) != 3 {
		t.Fatalf("lines = %#v", lines)
	}
}

func TestFormatTurnMetricsWrapByWidth(t *testing.T) {
	cfg := TurnMetricsUIConfig{Fields: []string{"total", "output_tokens", "tokens_per_sec", "tools", "cost"}}
	cfg.normalize()
	lines := FormatTurnMetricsLines(TurnMetricsValues{
		TotalDurationMs:  18000,
		OutputTokens:     128,
		TokensPerSec:     38.5,
		ToolCalls:        2,
		EstimatedCostUSD: 0.003,
	}, cfg, 28)
	if len(lines) < 2 {
		t.Fatalf("expected wrap, got %#v", lines)
	}
}
