package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/session"
)

type SwitchAgentMode struct{}

func (SwitchAgentMode) Name() string { return "switch_agent_mode" }

func (SwitchAgentMode) Description() string {
	return "Switch between agent mode (tools enabled) and plan mode (no tools; planning text only). Takes effect on the next model step."
}

func (SwitchAgentMode) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"mode": map[string]any{
				"type":        "string",
				"description": "agent | plan",
			},
		},
		"required": []string{"mode"},
	}
}

func (SwitchAgentMode) Run(ctx context.Context, args map[string]any) (string, error) {
	sess, ok := session.ActiveSessionFrom(ctx)
	if !ok {
		return "", fmt.Errorf("no active session")
	}
	mode := strings.ToLower(strings.TrimSpace(stringArg(args, "mode")))
	switch mode {
	case "agent", "plan":
	default:
		return "", fmt.Errorf("mode must be agent or plan")
	}
	sess.State.Settings.InteractionMode = mode
	if err := sess.Save(); err != nil {
		return "", err
	}
	return "interaction mode set to " + mode, nil
}
