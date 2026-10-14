package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

const decideToolName = "decide"

func emitDecisionToolActivity(
	emit func(events.Event),
	step int,
	tc provider.ToolCall,
	outcome toolExecOutcome,
) {
	out := outcome.text
	data := map[string]any(nil)
	if !outcome.duplicate {
		out = outcome.result.Text
		data = outcome.result.Data
		if outcome.err != nil {
			out = "error: " + outcome.err.Error()
		}
	}

	emit(events.Event{
		Type:       events.TypeToolCall,
		Step:       step,
		ToolName:   tc.Name,
		ToolID:     tc.ID,
		Args:       tc.Args,
		IsDecision: true,
	})
	emit(events.Event{
		Type:       events.TypeObservation,
		Step:       step,
		ToolName:   tc.Name,
		ToolID:     tc.ID,
		Text:       out,
		DurationMs: outcome.durationMs,
		IsDecision: true,
		Data:       data,
	})
}

func isDecisionTool(name string) bool {
	return strings.TrimSpace(name) == decideToolName
}
