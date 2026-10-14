package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func emitAskUserToolActivity(
	emit func(events.Event),
	step int,
	sessionID string,
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
		SessionID:  sessionID,
		ToolName:   tc.Name,
		ToolID:     tc.ID,
		Args:       tc.Args,
		IsAskUser:  true,
		Data:       data,
	})
	emit(events.Event{
		Type:       events.TypeObservation,
		Step:       step,
		SessionID:  sessionID,
		ToolName:   tc.Name,
		ToolID:     tc.ID,
		Text:       out,
		DurationMs: outcome.durationMs,
		IsAskUser:  true,
		Data:       data,
	})
}

func isAskUserTool(name string) bool {
	return strings.TrimSpace(name) == builtin.AskUserToolName()
}
