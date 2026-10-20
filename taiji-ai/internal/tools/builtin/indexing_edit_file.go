package builtin

import (
	"context"
	"strings"

	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/tools"
)

// IndexingEditFile wraps edit_file and auto-indexes text files via Taiji RAG scope.
type IndexingEditFile struct {
	inner EditFile
	svc   *rag.Service
}

func NewIndexingEditFile(svc *rag.Service) IndexingEditFile {
	return IndexingEditFile{svc: svc}
}

func (t IndexingEditFile) Name() string { return t.inner.Name() }

func (t IndexingEditFile) Description() string {
	return t.inner.Description() + " Text files are auto-indexed into the active RAG scope after a successful edit."
}

func (t IndexingEditFile) Parameters() map[string]any { return t.inner.Parameters() }

func (t IndexingEditFile) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := t.RunWithMetadata(ctx, args)
	return res.Text, err
}

func (t IndexingEditFile) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	res, err := t.inner.RunWithMetadata(ctx, args)
	if err != nil || t.svc == nil {
		return res, err
	}
	paths := editFileIndexPaths(args)
	for _, path := range paths {
		if path == "" || !rag.ShouldAutoIndexPath(path) {
			continue
		}
		indexRes, indexErr := rag.AutoIndexFile(ctx, t.svc, path)
		if note := rag.FormatAutoIndexNote(indexRes, indexErr); note != "" {
			res.Text += "\n" + note
		}
	}
	return res, nil
}

func editFileIndexPaths(args map[string]any) []string {
	if raw, ok := args["files"].([]any); ok {
		var paths []string
		for _, item := range raw {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if p, ok := m["path"].(string); ok && strings.TrimSpace(p) != "" {
				paths = append(paths, strings.TrimSpace(p))
			}
		}
		if len(paths) > 0 {
			return paths
		}
	}
	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path != "" {
		return []string{path}
	}
	return nil
}
