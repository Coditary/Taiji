package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileOutline struct{}

func (FileOutline) Name() string { return "file_outline" }

func (FileOutline) Description() string {
	return "List top-level symbols in a source file with line ranges. Go files use go/ast; other files use lightweight header heuristics. Use before read_file on large files."
}

func (FileOutline) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute or relative file path",
			},
		},
		"required": []string{"path"},
	}
}

func (FileOutline) Run(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".go" {
		out, err := outlineGoSource(path, raw)
		if err != nil {
			return "", err
		}
		return out, nil
	}
	return outlineHeuristic(raw), nil
}
