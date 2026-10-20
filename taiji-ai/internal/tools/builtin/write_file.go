package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coditary/taiji/internal/diff"
	"github.com/coditary/taiji/internal/tools"
)

const writeFileDiffMaxLines = 20

type WriteFile struct{}

func (WriteFile) Name() string { return "write_file" }

func (WriteFile) Description() string {
	return "Write UTF-8 text to a file, creating parent directories if needed."
}

func (WriteFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute or relative file path",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "File content",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (WriteFile) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := WriteFile{}.RunWithMetadata(ctx, args)
	return res.Text, err
}

func (WriteFile) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return tools.RunResult{}, fmt.Errorf("path is required")
	}
	if err := CaptureWorkspaceBaseline(ctx, path); err != nil {
		return tools.RunResult{}, err
	}

	oldContent, isNew, readErr := readExistingFile(path)
	if readErr != nil {
		return tools.RunResult{}, readErr
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return tools.RunResult{}, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return tools.RunResult{}, err
	}
	NoteWorkspaceModified(ctx, path)

	text := fmt.Sprintf("wrote %d bytes to %s", len(content), path)
	data := map[string]any{
		"kind":   "file_write",
		"path":   path,
		"bytes":  len(content),
		"is_new": isNew,
	}
	if unified := diff.Unified(path, oldContent, content, writeFileDiffMaxLines); unified != "" {
		data["diff"] = unified
	}
	text = enrichResultWithGoDiagnostics(path, text)
	return tools.RunResult{Text: text, Data: data}, nil
}

func readExistingFile(path string) (string, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", true, nil
		}
		return "", false, err
	}
	return string(raw), false, nil
}
