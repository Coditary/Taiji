package agent

import (
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestIsAskUserTool(t *testing.T) {
	if !isAskUserTool(builtin.AskUserToolName()) {
		t.Fatal("ask_user")
	}
	if isAskUserTool("list_dir") {
		t.Fatal("not ask_user")
	}
}

func TestEmitAskUserToolActivity(t *testing.T) {
	var got []events.Event
	emit := func(ev events.Event) { got = append(got, ev) }
	tc := provider.ToolCall{ID: "1", Name: builtin.AskUserToolName(), Args: `{}`}
	emitAskUserToolActivity(emit, 2, "sess", tc, toolExecOutcome{
		result: tools.RunResult{Text: "answered", Data: map[string]any{"ok": true}},
	})
	if len(got) != 2 || !got[0].IsAskUser || !got[1].IsAskUser {
		t.Fatalf("events=%+v", got)
	}
	emitAskUserToolActivity(emit, 3, "sess", tc, toolExecOutcome{
		duplicate: true,
		text:      "dup",
		err:       errString("boom"),
	})
	if len(got) != 4 {
		t.Fatalf("len=%d", len(got))
	}
}

type errString string

func (e errString) Error() string { return string(e) }
