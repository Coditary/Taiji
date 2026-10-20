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

type BatchEdit struct{}

func (BatchEdit) Name() string { return "batch_edit" }

func (BatchEdit) Description() string {
	return "Apply edits to multiple files atomically. All old_text matches are validated before any write. gofmt runs per file; go vet runs once per affected package at the end."
}

func (BatchEdit) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"files": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path":  map[string]any{"type": "string"},
						"edits": map[string]any{"type": "array"},
					},
					"required": []string{"path", "edits"},
				},
			},
		},
		"required": []string{"files"},
	}
}

func (BatchEdit) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := (BatchEdit{}).RunWithMetadata(ctx, args)
	return res.Text, err
}

func (BatchEdit) HideFromCatalog() bool { return true }

func (BatchEdit) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	return runBatchEdit(ctx, args)
}

func runBatchEdit(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	entries, err := parseBatchEditFiles(args)
	if err != nil {
		return tools.RunResult{}, err
	}
	planned := make([]batchEditPlan, 0, len(entries))
	for _, e := range entries {
		oldContent, isNew, readErr := readExistingFile(e.path)
		if readErr != nil {
			return tools.RunResult{}, readErr
		}
		if isNew {
			return tools.RunResult{}, fmt.Errorf("file does not exist: %s", e.path)
		}
		newContent, _, err := applyFileEdits(oldContent, e.edits)
		if err != nil {
			return tools.RunResult{}, fmt.Errorf("%s: %w", e.path, err)
		}
		planned = append(planned, batchEditPlan{
			path:       e.path,
			oldContent: oldContent,
			newContent: newContent,
			edits:      len(e.edits),
		})
	}
	for _, p := range planned {
		if err := CaptureWorkspaceBaseline(ctx, p.path); err != nil {
			return tools.RunResult{}, err
		}
	}
	var blocks []string
	written := make([]string, 0, len(planned))
	for _, p := range planned {
		if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil && filepath.Dir(p.path) != "." {
			return tools.RunResult{}, err
		}
		if err := os.WriteFile(p.path, []byte(p.newContent), 0o644); err != nil {
			return tools.RunResult{}, err
		}
		NoteWorkspaceModified(ctx, p.path)
		written = append(written, p.path)
		if toolsSettings.AutoFormatOrDefault() {
			_ = runGofmt(p.path)
		}
		u := diff.Unified(p.path, p.oldContent, p.newContent, editFileDiffMaxLines)
		block := fmt.Sprintf("%s (%d edit(s))", p.path, p.edits)
		if u != "" {
			block += "\n" + u
		}
		if msg := goSyntaxDiagnostics(p.path); msg != "" {
			block += "\n" + msg
		}
		blocks = append(blocks, block)
	}
	text := fmt.Sprintf("batch_edit applied to %d file(s)\n\n%s", len(planned), strings.Join(blocks, "\n\n"))
	if note := runDeferredGoVet(written); note != "" {
		text += "\n\n" + note
	}
	return tools.RunResult{Text: text, Data: map[string]any{"kind": "batch_edit", "files": len(planned)}}, nil
}

type batchEditPlan struct {
	path       string
	oldContent string
	newContent string
	edits      int
}

type batchFileEdits struct {
	path  string
	edits []fileEdit
}

func parseBatchEditFiles(args map[string]any) ([]batchFileEdits, error) {
	raw, ok := args["files"]
	if !ok {
		return nil, fmt.Errorf("files is required")
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("files must be an array")
	}
	out := make([]batchFileEdits, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("files[%d] must be an object", i)
		}
		path := strings.TrimSpace(stringArg(m, "path"))
		if path == "" {
			return nil, fmt.Errorf("files[%d]: path is required", i)
		}
		edits, err := parseFileEdits(m)
		if err != nil {
			return nil, fmt.Errorf("files[%d]: %w", i, err)
		}
		out = append(out, batchFileEdits{path: path, edits: edits})
	}
	return out, nil
}

func runDeferredGoVet(paths []string) string {
	if !toolsSettings.AutoVetOrDefault() {
		return ""
	}
	seen := map[string]bool{}
	var parts []string
	for _, p := range paths {
		if !strings.HasSuffix(strings.ToLower(p), ".go") {
			continue
		}
		dir := filepath.Dir(p)
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
		if seen[dir] {
			continue
		}
		seen[dir] = true
		if out := runGoVetPackage(filepath.Join(dir, "x.go")); out != "" {
			parts = append(parts, out)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "[DIAGNOSTICS WARNING]\n" + strings.Join(parts, "\n")
}
