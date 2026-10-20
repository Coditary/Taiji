package builtin

import (
	"context"
	"fmt"
	"strings"
)

type WorkspaceCheckpoint struct{}

func (WorkspaceCheckpoint) Name() string { return "workspace_checkpoint" }

func (WorkspaceCheckpoint) Description() string {
	return "Session-bound workspace snapshots (non-git). Actions: create (label), list, diff (optional checkpoint_id or path; defaults to git diff HEAD), restore (checkpoint_id, optional paths)."
}

func (WorkspaceCheckpoint) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "create | list | diff | restore",
			},
			"label": map[string]any{
				"type":        "string",
				"description": "Checkpoint label (create)",
			},
			"checkpoint_id": map[string]any{
				"type":        "string",
				"description": "Checkpoint id (diff, restore)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Single file path for diff",
			},
			"paths": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Subset of paths to restore",
			},
		},
		"required": []string{"action"},
	}
}

func (WorkspaceCheckpoint) Run(ctx context.Context, args map[string]any) (string, error) {
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	sessionID := ToolSessionIDFrom(ctx)
	mgr := DefaultWorkspace()

	switch action {
	case "create":
		id, summary, err := mgr.CreateCheckpoint(sessionID, stringArg(args, "label"))
		if err != nil {
			return "", err
		}
		return summary + "\n(id: " + id + ")", nil
	case "list":
		return mgr.ListCheckpoints(sessionID)
	case "diff":
		return mgr.Diff(sessionID, stringArg(args, "checkpoint_id"), stringArg(args, "path"))
	case "restore":
		cpID := strings.TrimSpace(stringArg(args, "checkpoint_id"))
		if cpID == "" {
			return "", fmt.Errorf("checkpoint_id is required for restore")
		}
		var paths []string
		if raw, ok := args["paths"]; ok {
			switch v := raw.(type) {
			case []string:
				paths = v
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						paths = append(paths, s)
					}
				}
			}
		}
		return mgr.Restore(ctx, sessionID, cpID, paths)
	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
}

// CaptureWorkspaceBaseline records the pre-session file state before the first write.
func CaptureWorkspaceBaseline(ctx context.Context, path string) error {
	return DefaultWorkspace().CapturePreWrite(ctx, ToolSessionIDFrom(ctx), path)
}

// NoteWorkspaceModified marks a path as changed in the current session.
func NoteWorkspaceModified(ctx context.Context, path string) {
	DefaultWorkspace().noteModified(ToolSessionIDFrom(ctx), path)
}
