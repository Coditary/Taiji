package agent

import (
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
)

func TestEmitDecisionToolActivity(t *testing.T) {
	var got []events.Event
	emit := func(ev events.Event) { got = append(got, ev) }
	tc := provider.ToolCall{ID: "1", Name: "decide", Args: `{}`}
	emitDecisionToolActivity(emit, 1, tc, toolExecOutcome{
		result: tools.RunResult{Text: "chosen"},
	})
	if len(got) != 2 || got[0].Type != events.TypeToolCall {
		t.Fatalf("events=%+v", got)
	}
}
