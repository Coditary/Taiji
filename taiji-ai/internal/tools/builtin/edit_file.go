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

type EditFile struct{}

func (EditFile) Name() string { return "edit_file" }

func (EditFile) Description() string {
	return "Search-and-replace edits on one file (path+edits) or many files atomically (files). Prefer over write_file for partial changes."
}

func (EditFile) Parameters() map[string]any {
	editItem := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"old_text": map[string]any{"type": "string"},
			"new_text": map[string]any{"type": "string"},
		},
		"required": []string{"old_text", "new_text"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Single-file path"},
			"edits": map[string]any{
				"type":  "array",
				"items": editItem,
			},
			"files": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path":  map[string]any{"type": "string"},
						"edits": map[string]any{"type": "array", "items": editItem},
					},
					"required": []string{"path", "edits"},
				},
				"description": "Multi-file batch (alternative to path+edits)",
			},
		},
	}
}

func (EditFile) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := (EditFile{}).RunWithMetadata(ctx, args)
	return res.Text, err
}

func (EditFile) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	if _, hasFiles := args["files"]; hasFiles {
		return runBatchEdit(ctx, args)
	}
	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return tools.RunResult{}, fmt.Errorf("path and edits are required (or use files for batch edits)")
	}
	if err := CaptureWorkspaceBaseline(ctx, path); err != nil {
		return tools.RunResult{}, err
	}
	edits, err := parseFileEdits(args)
	if err != nil {
		return tools.RunResult{}, err
	}

	oldContent, isNew, readErr := readExistingFile(path)
	if readErr != nil {
		return tools.RunResult{}, readErr
	}
	if isNew {
		return tools.RunResult{}, fmt.Errorf("file does not exist: %s", path)
	}

	newContent, changedLines, err := applyFileEdits(oldContent, edits)
	if err != nil {
		return tools.RunResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return tools.RunResult{}, err
	}
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return tools.RunResult{}, err
	}
	NoteWorkspaceModified(ctx, path)

	text := fmt.Sprintf("edited %s (%d edit(s))", path, len(edits))
	if len(changedLines) > 0 {
		text += fmt.Sprintf("; first change near line %d", changedLines[0])
	}
	unified := diff.Unified(path, oldContent, newContent, editFileDiffMaxLines)
	if unified != "" {
		text += "\n\n" + unified
	}

	data := map[string]any{
		"kind":          "file_edit",
		"path":          path,
		"edits":         len(edits),
		"changed_lines": changedLines,
		"is_new":        false,
	}
	if unified != "" {
		data["diff"] = unified
	}
	text = enrichResultWithGoDiagnostics(path, text)
	return tools.RunResult{Text: text, Data: data}, nil
}
