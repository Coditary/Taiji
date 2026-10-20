package builtin

import (
	"context"
	"fmt"
	"strings"
)

// TodoPlanTool tracks multi-step session work.
type TodoPlanTool struct{}

func (TodoPlanTool) Name() string { return "todo_plan" }

func (TodoPlanTool) Description() string {
	return "Track multi-step work in the current session. Actions: set (replace list), update (merge by id), get. Status: pending, in_progress, done, cancelled."
}

func (TodoPlanTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "set | update | get",
			},
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":     map[string]any{"type": "string"},
						"task":   map[string]any{"type": "string"},
						"status": map[string]any{"type": "string"},
					},
					"required": []string{"id"},
				},
			},
		},
		"required": []string{"action"},
	}
}

func (TodoPlanTool) Run(ctx context.Context, args map[string]any) (string, error) {
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	sessionID := ToolSessionIDFrom(ctx)
	mgr := DefaultTodoManager()
	switch action {
	case "set":
		items, err := parseTodoItems(args)
		if err != nil {
			return "", err
		}
		return mgr.Set(sessionID, items)
	case "update":
		items, err := parseTodoItems(args)
		if err != nil {
			return "", err
		}
		return mgr.Update(sessionID, items)
	case "get":
		return mgr.Get(sessionID), nil
	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
}
