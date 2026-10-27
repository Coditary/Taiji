package builtin

import "context"

// DeleteFile removes a single file (Cursor-style alias over manage_file delete).
type DeleteFile struct{}

func (DeleteFile) Name() string { return "delete_file" }

func (DeleteFile) Description() string {
	return "Delete a file in the workspace (checkpoint baselines preserved). Prefer manage_file when moving files."
}

func (DeleteFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path to delete",
			},
		},
		"required": []string{"path"},
	}
}

func (DeleteFile) Run(ctx context.Context, args map[string]any) (string, error) {
	args["action"] = "delete"
	return (ManageFile{}).Run(ctx, args)
}
