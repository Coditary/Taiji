package builtin

import (
	"context"
	"strings"

	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/tools"
)

// IndexingWriteFile wraps write_file and auto-indexes text files via Taiji RAG scope.
type IndexingWriteFile struct {
	inner WriteFile
	svc   *rag.Service
}

func NewIndexingWriteFile(svc *rag.Service) IndexingWriteFile {
	return IndexingWriteFile{svc: svc}
}

func (t IndexingWriteFile) Name() string { return t.inner.Name() }

func (t IndexingWriteFile) Description() string {
	return t.inner.Description() + " Text files are auto-indexed into the active RAG scope after a successful write."
}

func (t IndexingWriteFile) Parameters() map[string]any { return t.inner.Parameters() }

func (t IndexingWriteFile) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := t.RunWithMetadata(ctx, args)
	return res.Text, err
}

func (t IndexingWriteFile) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	res, err := t.inner.RunWithMetadata(ctx, args)
	if err != nil || t.svc == nil {
		return res, err
	}
	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" || !rag.ShouldAutoIndexPath(path) {
		return res, nil
	}
	indexRes, indexErr := rag.AutoIndexFile(ctx, t.svc, path)
	if note := rag.FormatAutoIndexNote(indexRes, indexErr); note != "" {
		res.Text += "\n" + note
	}
	return res, nil
}
