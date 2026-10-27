package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EditNotebook struct{}

func (EditNotebook) Name() string { return "edit_notebook" }

func (EditNotebook) Description() string {
	return "Edit Jupyter notebook cells (.ipynb). Cell indices are 0-based. Use is_new_cell to insert or replace cell source."
}

func (EditNotebook) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target_notebook": map[string]any{"type": "string"},
			"cell_idx":        map[string]any{"type": "integer"},
			"is_new_cell":     map[string]any{"type": "boolean"},
			"cell_language": map[string]any{
				"type":        "string",
				"description": "python, markdown, javascript, typescript, r, sql, shell, raw, other",
			},
			"old_string": map[string]any{"type": "string"},
			"new_string": map[string]any{"type": "string"},
		},
		"required": []string{"target_notebook", "cell_idx", "is_new_cell", "cell_language", "old_string", "new_string"},
	}
}

func (EditNotebook) Run(ctx context.Context, args map[string]any) (string, error) {
	path := strings.TrimSpace(stringArg(args, "target_notebook"))
	if path == "" {
		return "", fmt.Errorf("target_notebook is required")
	}
	abs, err := explorePath(path)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(abs), ".ipynb") {
		return "", fmt.Errorf("not a .ipynb notebook: %s", abs)
	}
	cellIdx := intArg(args, "cell_idx")
	isNew, _ := args["is_new_cell"].(bool)
	lang := strings.TrimSpace(stringArg(args, "cell_language"))
	oldStr := stringArg(args, "old_string")
	newStr := stringArg(args, "new_string")

	var nb map[string]any
	if isNew && cellIdx == 0 && !fileExists(abs) {
		nb = newNotebook()
	} else {
		raw, err := os.ReadFile(abs)
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal(raw, &nb); err != nil {
			return "", fmt.Errorf("parse notebook: %w", err)
		}
	}
	cells, err := notebookCells(nb)
	if err != nil {
		return "", err
	}
	if isNew {
		if cellIdx < 0 || cellIdx > len(cells) {
			return "", fmt.Errorf("cell_idx out of range")
		}
		cell := newNotebookCell(lang, newStr)
		insert := append([]map[string]any{cell}, cells[cellIdx:]...)
		cells = append(cells[:cellIdx], insert...)
	} else {
		if cellIdx < 0 || cellIdx >= len(cells) {
			return "", fmt.Errorf("cell_idx out of range")
		}
		src, err := cellSource(cells[cellIdx])
		if err != nil {
			return "", err
		}
		if oldStr == "" && newStr != "" && src != "" {
			return "", fmt.Errorf("old_string must match existing cell content for edits")
		}
		if oldStr != "" && !strings.Contains(src, oldStr) {
			return "", fmt.Errorf("old_string not found in cell")
		}
		updated := strings.Replace(src, oldStr, newStr, 1)
		setCellSource(cells[cellIdx], updated)
	}
	nb["cells"] = cells
	out, err := json.MarshalIndent(nb, "", " ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, out, 0o644); err != nil {
		return "", err
	}
	NoteWorkspaceModified(ctx, abs)
	return fmt.Sprintf("updated notebook %s (cells=%d)", abs, len(cells)), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func newNotebook() map[string]any {
	return map[string]any{
		"nbformat":       4,
		"nbformat_minor": 5,
		"metadata":       map[string]any{},
		"cells":          []map[string]any{},
	}
}

func notebookCells(nb map[string]any) ([]map[string]any, error) {
	raw, ok := nb["cells"].([]any)
	if !ok {
		return nil, fmt.Errorf("notebook missing cells")
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid cell entry")
		}
		out = append(out, m)
	}
	return out, nil
}

func newNotebookCell(lang, source string) map[string]any {
	cellType := "code"
	nbLang := mapLanguage(lang)
	if nbLang == "markdown" {
		cellType = "markdown"
	}
	cell := map[string]any{
		"cell_type": cellType,
		"metadata":  map[string]any{},
		"source":    splitLines(source),
	}
	if cellType == "code" {
		cell["execution_count"] = nil
		cell["outputs"] = []any{}
	}
	return cell
}

func mapLanguage(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "markdown", "md":
		return "markdown"
	case "python", "javascript", "typescript", "r", "sql", "shell", "raw", "other":
		return strings.ToLower(lang)
	default:
		return "python"
	}
}

func cellSource(cell map[string]any) (string, error) {
	switch src := cell["source"].(type) {
	case nil:
		return "", nil
	case string:
		return src, nil
	case []any:
		var b strings.Builder
		for _, line := range src {
			if s, ok := line.(string); ok {
				b.WriteString(s)
			}
		}
		return b.String(), nil
	default:
		return "", fmt.Errorf("unsupported cell source type")
	}
}

func setCellSource(cell map[string]any, source string) {
	cell["source"] = splitLines(source)
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, "\n")
	for i := range parts {
		parts[i] += "\n"
	}
	return parts
}
