package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type WorkspaceFind struct{}

func (WorkspaceFind) Name() string { return "workspace_find" }

func (WorkspaceFind) Description() string {
	return "Fuzzy-find files and project markers under a directory (ignore-aware walk). Use to locate projects by name (e.g. music app under ~/Dev)."
}

func (WorkspaceFind) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Substring to match in relative paths (e.g. music, my-app)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Root directory (default: project root or home)",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Max paths (default 50)",
			},
		},
		"required": []string{"query"},
	}
}

func (WorkspaceFind) Run(ctx context.Context, args map[string]any) (string, error) {
	query := strings.TrimSpace(stringArg(args, "query"))
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	root := strings.TrimSpace(stringArg(args, "path"))
	if root == "" {
		root = workspaceProjectRoot()
		if root == "" || root == "." {
			if home, err := os.UserHomeDir(); err == nil {
				root = home
			}
		}
	}
	abs, err := explorePath(root)
	if err != nil {
		return "", err
	}
	max := intArg(args, "max_results")
	if max <= 0 {
		max = 50
	}
	lines, err := runTaijiToolsLines(ctx, "find", "--root", abs, "--query", query, "--max", fmt.Sprintf("%d", max))
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return fmt.Sprintf("no paths matching %q under %s", query, abs), nil
	}
	return strings.Join(lines, "\n"), nil
}
