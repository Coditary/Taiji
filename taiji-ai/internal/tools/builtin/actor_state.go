package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/session"
)

// UpdateActorState mutates session actor_state for the active chat.
type UpdateActorState struct{}

func (UpdateActorState) Name() string { return "update_actor_state" }

func (UpdateActorState) Description() string {
	return "Update a key in the chat actor state (shown in the TUI sidebar). Use only keys from the active template state_schema."
}

func (UpdateActorState) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"key":   map[string]any{"type": "string", "description": "State key"},
			"value": map[string]any{"type": "string", "description": "New value"},
		},
		"required": []string{"key", "value"},
	}
}

func (UpdateActorState) Run(ctx context.Context, args map[string]any) (string, error) {
	sess, ok := session.ActiveSessionFrom(ctx)
	if !ok {
		return "", fmt.Errorf("no active session")
	}
	key := strings.TrimSpace(stringArg(args, "key"))
	value := strings.TrimSpace(stringArg(args, "value"))
	if key == "" {
		return "", fmt.Errorf("key is required")
	}
	if sess.State.Settings.ActorState == nil {
		sess.State.Settings.ActorState = map[string]string{}
	}
	sess.State.Settings.ActorState[key] = value
	if err := sess.Save(); err != nil {
		return "", err
	}
	return fmt.Sprintf("state[%s]=%s", key, value), nil
}
