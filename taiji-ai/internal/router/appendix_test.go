package router

import (
	"strings"
	"testing"
)

func TestRouteTurnControlVariants(t *testing.T) {
	for _, msg := range []string{"abort", "quit", "exit", "/stop now"} {
		plan := RouteTurn(msg)
		if plan.Action != ActionStop {
			t.Fatalf("msg=%q action=%s", msg, plan.Action)
		}
	}
	plan := RouteTurn("!hook deploy")
	if plan.Action != ActionHook {
		t.Fatalf("action=%s", plan.Action)
	}
}

func TestSystemPromptAppendixNonEmpty(t *testing.T) {
	text := SystemPromptAppendix()
	if !strings.Contains(text, "Turn routing") || !strings.Contains(text, "unknown") {
		t.Fatalf("appendix=%q", text)
	}
}

func TestPlanAppendixUnknownWithSkill(t *testing.T) {
	plan := Plan{
		Action:      ActionUnknown,
		Reason:      "test reason",
		SkillName:   "deploy",
		SkillMatches: []SkillMatch{{Name: "deploy"}, {Name: "lint"}},
	}
	text := PlanAppendix(plan)
	if !strings.Contains(text, "unknown") || !strings.Contains(text, "deploy") {
		t.Fatalf("text=%q", text)
	}
}

func TestPlanAppendixStopAndEmpty(t *testing.T) {
	stop := PlanAppendix(Plan{Action: ActionStop, Reason: "user stop"})
	if !strings.Contains(stop, "Stop processing") {
		t.Fatalf("stop=%q", stop)
	}
	if PlanAppendix(Plan{}) != "" {
		t.Fatal("empty action should yield empty appendix")
	}
}
