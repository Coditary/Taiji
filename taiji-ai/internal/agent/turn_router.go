package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/router"
)

const turnRouteToolName = "turn_route"

type turnRouterResult struct {
	plan         router.Plan
	goal         string
	prefetch     memoryPrefetchResult
	skillMatches []router.SkillMatch
	stopAnswer   string
}

func runTurnRouter(
	ctx context.Context,
	req RunRequest,
	goal string,
	emit func(events.Event),
) turnRouterResult {
	plan := router.RouteTurn(goal)

	out := turnRouterResult{plan: plan, goal: goal}
	if plan.Action == router.ActionStop {
		out.stopAnswer = strings.TrimSpace(plan.Message)
		if out.stopAnswer == "" {
			out.stopAnswer = "Gestoppt."
		}
		emitTurnRoute(plan, emit)
		return out
	}

	if req.Config != nil && req.Config.ResolvedMemory().PrefetchOnTurnOrDefault() && req.Memory != nil && req.Memory.Enabled() {
		goal, out.prefetch = runMemoryPrefetch(ctx, req, goal, emit)
		out.goal = goal
	}

	out.skillMatches = runSkillPrefetch(ctx, req, goal, emit)
	plan.SkillMatches = out.skillMatches
	if len(out.skillMatches) > 0 && strings.TrimSpace(plan.SkillName) == "" {
		if top := out.skillMatches[0]; top.Fit == "high" || top.Fit == "medium" {
			plan.SkillName = top.Name
		}
	}
	if preface := runCodePrefetch(ctx, req, goal, emit); preface != "" {
		out.goal = strings.TrimSpace(preface + "\n\n" + goal)
	}
	out.plan = plan
	emitTurnRoute(plan, emit)
	return out
}

func emitTurnRoute(plan router.Plan, emit func(events.Event)) {
	if emit == nil {
		return
	}
	args, _ := json.Marshal(map[string]any{
		"action":         plan.Action,
		"reason":         plan.Reason,
		"memory_kind":    plan.MemoryKind,
		"skill_name":     plan.SkillName,
		"skill_matches":  plan.SkillMatches,
		"decide_profile": plan.DecideProfile,
		"tool_hint":      plan.ToolHint,
	})
	emit(events.Event{
		Type:     events.TypeToolCall,
		Step:     1,
		ToolName: turnRouteToolName,
		Args:     string(args),
		Text:     "route",
	})
	emit(events.Event{
		Type:       events.TypeObservation,
		Step:       1,
		ToolName:   turnRouteToolName,
		Text:       formatTurnRouteObservation(plan),
		DurationMs: 0,
	})
}

func formatTurnRouteObservation(plan router.Plan) string {
	line := fmt.Sprintf("Routing → %s", plan.Action)
	if kind := strings.TrimSpace(plan.MemoryKind); kind != "" {
		line += fmt.Sprintf(" (kind: %s)", kind)
	}
	if hint := strings.TrimSpace(plan.ToolHint); hint != "" {
		line += fmt.Sprintf(" (tool: %s)", hint)
	}
	if skill := strings.TrimSpace(plan.SkillName); skill != "" {
		line += fmt.Sprintf(" (skill: %s)", skill)
	}
	if len(plan.SkillMatches) > 0 {
		line += fmt.Sprintf("\nTop skill matches: %d", len(plan.SkillMatches))
		for i, m := range plan.SkillMatches {
			if i >= 3 {
				break
			}
			line += fmt.Sprintf("\n- %s (fit: %s, %.2f)", m.Name, m.Fit, m.Score)
		}
	}
	if reason := strings.TrimSpace(plan.Reason); reason != "" {
		line += "\n" + reason
	}
	return line
}
