package agent

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/router"
)

func TestRunTurnRouterEmitsRouteBeforeSkillSearch(t *testing.T) {
	var kinds []string
	routed := runTurnRouter(context.Background(), RunRequest{}, "List files in the current directory", func(ev events.Event) {
		kinds = append(kinds, string(ev.Type)+":"+ev.ToolName)
	})

	if routed.plan.Action != router.ActionUnknown {
		t.Fatalf("plan=%s want unknown", routed.plan.Action)
	}
	if len(kinds) < 2 || kinds[0] != "tool_call:turn_route" {
		t.Fatalf("expected route event first, got %v", kinds)
	}
	if routed.prefetch.stored {
		t.Fatal("expected no keyword-driven memory prefetch")
	}
}

func TestRunTurnRouterStop(t *testing.T) {
	routed := runTurnRouter(context.Background(), RunRequest{}, "stop", func(events.Event) {})
	if routed.plan.Action != router.ActionStop || routed.stopAnswer == "" {
		t.Fatalf("stop result=%+v", routed)
	}
}
