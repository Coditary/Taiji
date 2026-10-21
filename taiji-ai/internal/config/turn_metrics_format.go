package config

import (
	"fmt"
	"strings"
)

// TurnMetricsValues is the numeric input for formatting a turn metrics line.
type TurnMetricsValues struct {
	TotalDurationMs      int64
	WaitDurationMs       int64
	ThinkingDurationMs   int64
	GenerationDurationMs int64
	OutputTokens         int
	InputTokens          int
	EstimatedCostUSD     float64
	TokensPerSec         float64
	HadThinking          bool
	ToolCalls            int
	ToolDurationMs       int64
	Steps                int
	ContextTokens        int
	ContextWindow        int
	ContextUsedPct       float64
}

// FormatTurnMetrics builds a single-line status string (legacy helper).
func FormatTurnMetrics(values TurnMetricsValues, cfg TurnMetricsUIConfig, locale string) string {
	lines := FormatTurnMetricsLines(values, cfg, 0, locale)
	return strings.Join(lines, "\n")
}

// FormatTurnMetricsLines builds wrapped status lines under an assistant reply.
func FormatTurnMetricsLines(values TurnMetricsValues, cfg TurnMetricsUIConfig, maxWidth int, locale string) []string {
	if !cfg.EnabledOrDefault() {
		return nil
	}

	labels := TurnMetricsLabelsForLocale(locale)
	var parts []string
	for _, field := range cfg.ResolvedFields() {
		if part := formatTurnMetricField(field, values, cfg.CompactOrDefault(), labels); part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return nil
	}

	sep := cfg.SeparatorOrDefault()
	if maxWidth > 0 {
		return wrapMetricPartsByWidth(parts, sep, maxWidth)
	}
	return wrapMetricPartsByCount(parts, sep, cfg.MaxFieldsPerLineOrDefault())
}

func wrapMetricPartsByCount(parts []string, sep string, maxFields int) []string {
	if maxFields <= 0 {
		maxFields = len(parts)
	}
	var lines []string
	for i := 0; i < len(parts); i += maxFields {
		end := i + maxFields
		if end > len(parts) {
			end = len(parts)
		}
		lines = append(lines, strings.Join(parts[i:end], sep))
	}
	return lines
}

func wrapMetricPartsByWidth(parts []string, sep string, maxWidth int) []string {
	if maxWidth <= 0 {
		return wrapMetricPartsByCount(parts, sep, len(parts))
	}
	sepWidth := displayWidth(sep)
	var lines []string
	var current []string
	currentWidth := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		lines = append(lines, strings.Join(current, sep))
		current = nil
		currentWidth = 0
	}

	for _, part := range parts {
		partWidth := displayWidth(part)
		addWidth := partWidth
		if len(current) > 0 {
			addWidth += sepWidth
		}
		if len(current) > 0 && currentWidth+addWidth > maxWidth {
			flush()
		}
		if len(current) == 0 {
			currentWidth = partWidth
		} else {
			currentWidth += sepWidth + partWidth
		}
		current = append(current, part)
	}
	flush()
	return lines
}

func displayWidth(text string) int {
	return len([]rune(text))
}

func formatTurnMetricField(field string, values TurnMetricsValues, compact bool, labels TurnMetricsLabels) string {
	switch field {
	case TurnMetricFieldTotal:
		if values.TotalDurationMs <= 0 {
			return ""
		}
		if compact {
			return formatDurationMs(values.TotalDurationMs)
		}
		return formatDurationMs(values.TotalDurationMs) + " " + labels.Total
	case TurnMetricFieldWait:
		if values.WaitDurationMs <= 0 {
			return ""
		}
		if compact {
			return formatDurationMs(values.WaitDurationMs)
		}
		return formatDurationMs(values.WaitDurationMs) + " " + labels.Wait
	case TurnMetricFieldThought:
		if !values.HadThinking || values.ThinkingDurationMs <= 0 {
			return ""
		}
		if compact {
			return formatDurationMs(values.ThinkingDurationMs)
		}
		return formatDurationMs(values.ThinkingDurationMs) + " " + labels.Thought
	case TurnMetricFieldGeneration:
		if values.GenerationDurationMs <= 0 {
			return ""
		}
		if compact {
			return formatDurationMs(values.GenerationDurationMs)
		}
		return formatDurationMs(values.GenerationDurationMs) + " " + labels.Generation
	case TurnMetricFieldOutputTokens:
		if values.OutputTokens <= 0 {
			return ""
		}
		if compact {
			return formatTokenCount(values.OutputTokens)
		}
		return formatTokenCount(values.OutputTokens) + " " + labels.Tok
	case TurnMetricFieldInputTokens:
		if values.InputTokens <= 0 {
			return ""
		}
		if compact {
			return formatTokenCount(values.InputTokens)
		}
		return formatTokenCount(values.InputTokens) + " " + labels.TokIn
	case TurnMetricFieldTokensPerSec:
		if values.TokensPerSec <= 0 {
			return ""
		}
		tokSuffix := strings.TrimSpace(strings.TrimPrefix(labels.TokPerSec, "%.1f"))
		if tokSuffix == labels.TokPerSec {
			tokSuffix = "tok/s"
		}
		if values.TokensPerSec >= 100 {
			return fmt.Sprintf("%.0f %s", values.TokensPerSec, tokSuffix)
		}
		return fmt.Sprintf(labels.TokPerSec, values.TokensPerSec)
	case TurnMetricFieldCost:
		if values.EstimatedCostUSD <= 0 {
			return ""
		}
		return formatCostUSD(values.EstimatedCostUSD)
	case TurnMetricFieldTools:
		if values.ToolCalls <= 0 {
			return ""
		}
		if values.ToolCalls == 1 {
			return labels.ToolOne
		}
		return fmt.Sprintf(labels.ToolsMany, values.ToolCalls)
	case TurnMetricFieldToolTime:
		if values.ToolDurationMs <= 0 {
			return ""
		}
		if compact {
			return formatDurationMs(values.ToolDurationMs)
		}
		return formatDurationMs(values.ToolDurationMs) + labels.ToolTime
	case TurnMetricFieldSteps:
		if values.Steps <= 0 {
			return ""
		}
		if values.Steps == 1 {
			return labels.StepOne
		}
		return fmt.Sprintf(labels.StepsMany, values.Steps)
	case TurnMetricFieldContext:
		if values.ContextTokens <= 0 {
			return ""
		}
		if values.ContextWindow > 0 {
			return fmt.Sprintf(labels.CtxPair, formatTokenCount(values.ContextTokens), formatTokenCount(values.ContextWindow))
		}
		return formatTokenCount(values.ContextTokens) + labels.Ctx
	case TurnMetricFieldContextPct:
		if values.ContextUsedPct <= 0 {
			return ""
		}
		return fmt.Sprintf(labels.CtxPct, values.ContextUsedPct)
	default:
		return ""
	}
}

func formatDurationMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	secs := float64(ms) / 1000.0
	if secs < 10 {
		return fmt.Sprintf("%.1fs", secs)
	}
	return fmt.Sprintf("%.0fs", secs)
}

func formatTokenCount(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 10_000 {
		return fmt.Sprintf("%.0fk", float64(n)/1_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func formatCostUSD(v float64) string {
	if v >= 1 {
		return fmt.Sprintf("$%.2f", v)
	}
	if v >= 0.01 {
		return fmt.Sprintf("$%.2f", v)
	}
	if v >= 0.001 {
		return fmt.Sprintf("$%.3f", v)
	}
	return fmt.Sprintf("$%.4f", v)
}
