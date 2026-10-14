package session

import (
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

// TurnsWithActivity builds chat turns and attaches persisted activity events per turn.
func TurnsWithActivity(messages []provider.Message, all []events.Event) []ChatTurn {
	turns := TurnsFromMessages(messages)
	if len(all) == 0 || len(turns) == 0 {
		return turns
	}

	groups := groupEventsByTurn(all)
	for i := range turns {
		if i >= len(groups) {
			break
		}
		turns[i].Activity = filterActivityEvents(groups[i])
		if m := metricsFromEvents(groups[i]); m != nil {
			turns[i].Metrics = m
		}
	}
	return turns
}

func metricsFromEvents(evts []events.Event) *TurnMetrics {
	for i := len(evts) - 1; i >= 0; i-- {
		ev := evts[i]
		if ev.Type != events.TypeTurnMetrics && ev.Type != events.TypeDone {
			continue
		}
		if ev.Type == events.TypeTurnMetrics {
			return metricsFromData(ev.Data)
		}
		if ev.Data != nil {
			if m := metricsFromData(ev.Data); m != nil {
				return m
			}
		}
	}
	return nil
}

func metricsFromData(data map[string]any) *TurnMetrics {
	if len(data) == 0 {
		return nil
	}
	m := &TurnMetrics{}
	if v, ok := data["total_duration_ms"]; ok {
		m.TotalDurationMs = toInt64(v)
	}
	if v, ok := data["wait_duration_ms"]; ok {
		m.WaitDurationMs = toInt64(v)
	}
	if v, ok := data["thinking_duration_ms"]; ok {
		m.ThinkingDurationMs = toInt64(v)
	}
	if v, ok := data["generation_duration_ms"]; ok {
		m.GenerationDurationMs = toInt64(v)
	}
	if v, ok := data["output_tokens"]; ok {
		m.OutputTokens = int(toInt64(v))
	}
	if v, ok := data["input_tokens"]; ok {
		m.InputTokens = int(toInt64(v))
	}
	if v, ok := data["estimated_cost_usd"]; ok {
		m.EstimatedCostUSD = toFloat64(v)
	}
	if v, ok := data["tokens_per_sec"]; ok {
		m.TokensPerSec = toFloat64(v)
	}
	if v, ok := data["had_thinking"]; ok {
		m.HadThinking = toBool(v)
	}
	if v, ok := data["tool_calls"]; ok {
		m.ToolCalls = int(toInt64(v))
	}
	if v, ok := data["tool_duration_ms"]; ok {
		m.ToolDurationMs = toInt64(v)
	}
	if v, ok := data["steps"]; ok {
		m.Steps = int(toInt64(v))
	}
	if v, ok := data["context_tokens"]; ok {
		m.ContextTokens = int(toInt64(v))
	}
	if v, ok := data["context_window"]; ok {
		m.ContextWindow = int(toInt64(v))
	}
	if v, ok := data["context_used_pct"]; ok {
		m.ContextUsedPct = toFloat64(v)
	}
	if m.TotalDurationMs <= 0 {
		return nil
	}
	return m
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}

func toBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	default:
		return false
	}
}

func groupEventsByTurn(all []events.Event) [][]events.Event {
	var groups [][]events.Event
	var current []events.Event
	seenDone := false

	flush := func() {
		if len(current) == 0 {
			return
		}
		groups = append(groups, current)
		current = nil
		seenDone = false
	}

	for _, ev := range all {
		if seenDone && ev.Type == events.TypeStep {
			flush()
		}
		current = append(current, ev)
		if ev.Type == events.TypeDone {
			seenDone = true
		}
	}
	flush()
	return groups
}

func filterActivityEvents(evts []events.Event) []events.Event {
	var out []events.Event
	for _, ev := range evts {
		switch ev.Type {
		case events.TypeToolCall, events.TypeObservation:
			out = append(out, ev)
		case events.TypeMemoryRecalled:
			if strings.TrimSpace(ev.ToolName) == "" || hasMatchingObservation(evts, ev) {
				continue
			}
			out = append(out, events.Event{
				Type:       events.TypeObservation,
				Step:       ev.Step,
				ToolName:   ev.ToolName,
				Text:       ev.Text,
				DurationMs: ev.DurationMs,
			})
		case events.TypeMemoryStored:
			if strings.TrimSpace(ev.Text) == "" {
				continue
			}
			out = append(out, events.Event{
				Type:       events.TypeObservation,
				Step:       ev.Step,
				ToolName:   "memory_remember",
				Text:       ev.Text,
				DurationMs: ev.DurationMs,
			})
		case events.TypeMemoryRevised:
			if strings.TrimSpace(ev.Text) == "" || hasMatchingObservation(evts, ev) {
				continue
			}
			out = append(out, events.Event{
				Type:       events.TypeObservation,
				Step:       ev.Step,
				ToolName:   "memory_revise",
				Text:       ev.Text,
				DurationMs: ev.DurationMs,
			})
		}
	}
	return out
}

func hasMatchingObservation(evts []events.Event, target events.Event) bool {
	targetText := strings.TrimSpace(target.Text)
	targetTool := strings.TrimSpace(target.ToolName)
	for _, ev := range evts {
		if ev.Type != events.TypeObservation {
			continue
		}
		if strings.TrimSpace(ev.ToolName) != targetTool && targetTool != "" {
			continue
		}
		if targetText != "" && strings.TrimSpace(ev.Text) == targetText {
			return true
		}
	}
	return false
}
