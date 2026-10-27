package builtin

import (
	"context"
	"fmt"

	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

// CreateChatTool registers a session-scoped script tool for later turns.
type CreateChatTool struct{}

func (CreateChatTool) Name() string { return "create_chat_tool" }

func (CreateChatTool) Description() string {
	return "Define a temporary session tool (python or shell). Args: name, description, language, script. The tool becomes callable on subsequent steps."
}

func (CreateChatTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":        map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
			"language":    map[string]any{"type": "string", "description": "python or shell"},
			"script":      map[string]any{"type": "string", "description": "Tool body (Python script or shell script)"},
		},
		"required": []string{"name", "language", "script"},
	}
}

func (CreateChatTool) Run(ctx context.Context, args map[string]any) (string, error) {
	s, ok := session.ActiveSessionFrom(ctx)
	if !ok {
		return "", fmt.Errorf("no active session")
	}
	tool := session.ChatTool{
		Name:        stringArg(args, "name"),
		Description: stringArg(args, "description"),
		Language:    stringArg(args, "language"),
		Script:      stringArg(args, "script"),
	}
	if err := s.AddChatTool(tool); err != nil {
		return "", err
	}
	return fmt.Sprintf("registered chat tool %q (%s)", tool.Name, tool.Language), nil
}

func EnableCreateChatTool(reg *tools.Registry) {
	if reg == nil {
		return
	}
	reg.Register(CreateChatTool{})
}
