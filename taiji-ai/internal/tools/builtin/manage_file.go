package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/tools"
)

type ManageFile struct{}

func (ManageFile) Name() string { return "manage_file" }

func (ManageFile) Description() string {
	return "Delete or move/rename files safely (workspace baselines preserved for checkpoint restore). Actions: delete, move."
}

func (ManageFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "delete | move",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Source file path",
			},
			"dest": map[string]any{
				"type":        "string",
				"description": "Destination path (move)",
			},
		},
		"required": []string{"action", "path"},
	}
}

func (ManageFile) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := (ManageFile{}).RunWithMetadata(ctx, args)
	return res.Text, err
}

func (ManageFile) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	path := strings.TrimSpace(stringArg(args, "path"))
	if path == "" {
		return tools.RunResult{}, fmt.Errorf("path is required")
	}
	switch action {
	case "delete":
		if err := CaptureWorkspaceBaseline(ctx, path); err != nil {
			return tools.RunResult{}, err
		}
		if err := os.Remove(path); err != nil {
			return tools.RunResult{}, err
		}
		NoteWorkspaceModified(ctx, path)
		reindexPath(ctx, path)
		return tools.RunResult{Text: "deleted " + path, Data: map[string]any{"kind": "file_delete", "path": path}}, nil
	case "move":
		dest := strings.TrimSpace(stringArg(args, "dest"))
		if dest == "" {
			return tools.RunResult{}, fmt.Errorf("dest is required for move")
		}
		if err := CaptureWorkspaceBaseline(ctx, path); err != nil {
			return tools.RunResult{}, err
		}
		if err := CaptureWorkspaceBaseline(ctx, dest); err != nil {
			return tools.RunResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil && filepath.Dir(dest) != "." {
			return tools.RunResult{}, err
		}
		if err := os.Rename(path, dest); err != nil {
			return tools.RunResult{}, err
		}
		NoteWorkspaceModified(ctx, path)
		NoteWorkspaceModified(ctx, dest)
		reindexPath(ctx, dest)
		return tools.RunResult{Text: fmt.Sprintf("moved %s -> %s", path, dest), Data: map[string]any{"kind": "file_move", "path": path, "dest": dest}}, nil
	default:
		return tools.RunResult{}, fmt.Errorf("unknown action %q", action)
	}
}

func reindexPath(ctx context.Context, path string) {
	svc := workspaceRAG()
	if svc == nil || !rag.ShouldAutoIndexPath(path) {
		return
	}
	_, _ = rag.AutoIndexFile(ctx, svc, path)
}
