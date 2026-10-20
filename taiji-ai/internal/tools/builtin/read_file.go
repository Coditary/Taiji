package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type ReadFile struct{}

func (ReadFile) Name() string { return "read_file" }

func (ReadFile) Description() string {
	return "Read a UTF-8 text file from disk. Lines are numbered (e.g. ' 120 | code'). For large files (>300 lines), only the first 200 lines are returned unless start_line/end_line are set."
}

func (ReadFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute or relative file path",
			},
			"start_line": map[string]any{
				"type":        "integer",
				"description": "First line to read (1-based, inclusive)",
			},
			"end_line": map[string]any{
				"type":        "integer",
				"description": "Last line to read (1-based, inclusive)",
			},
		},
		"required": []string{"path"},
	}
}

func (ReadFile) Run(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	start := intArg(args, "start_line")
	end := intArg(args, "end_line")
	out, _, _ := formatFileContentWithLineNumbers(string(data), start, end)
	return out, nil
}
