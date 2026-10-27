package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/session"
)

type CreateGoal struct{}

func (CreateGoal) Name() string { return "create_goal" }

func (CreateGoal) Description() string {
	return "Set a long-running objective for this chat (only when the user explicitly wants a tracked goal). Replaces any previous goal."
}

func (CreateGoal) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"objective": map[string]any{
				"type":        "string",
				"description": "Goal description",
			},
		},
		"required": []string{"objective"},
	}
}

func (CreateGoal) Run(ctx context.Context, args map[string]any) (string, error) {
	sess, ok := session.ActiveSessionFrom(ctx)
	if !ok {
		return "", fmt.Errorf("no active session")
	}
	objective := strings.TrimSpace(stringArg(args, "objective"))
	if objective == "" {
		return "", fmt.Errorf("objective is required")
	}
	sess.State.Settings.AgentGoalObjective = objective
	sess.State.Settings.AgentGoalStatus = "active"
	if err := sess.Save(); err != nil {
		return "", err
	}
	return "goal created (active)", nil
}

type UpdateGoal struct{}

func (UpdateGoal) Name() string { return "update_goal" }

func (UpdateGoal) Description() string {
	return "Update the tracked goal status: active, complete, or cancelled."
}

func (UpdateGoal) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{
				"type":        "string",
				"description": "active | complete | cancelled",
			},
		},
		"required": []string{"status"},
	}
}

func (UpdateGoal) Run(ctx context.Context, args map[string]any) (string, error) {
	sess, ok := session.ActiveSessionFrom(ctx)
	if !ok {
		return "", fmt.Errorf("no active session")
	}
	status := strings.ToLower(strings.TrimSpace(stringArg(args, "status")))
	switch status {
	case "active", "complete", "cancelled":
	default:
		return "", fmt.Errorf("status must be active, complete, or cancelled")
	}
	if strings.TrimSpace(sess.State.Settings.AgentGoalObjective) == "" {
		return "", fmt.Errorf("no goal exists; call create_goal first")
	}
	sess.State.Settings.AgentGoalStatus = status
	if err := sess.Save(); err != nil {
		return "", err
	}
	return fmt.Sprintf("goal status=%s", status), nil
}
