package builtin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/tools"
)

// EnableRAGOrchestration registers RAG tools and hooks write_file for scoped auto-indexing.
func EnableRAGOrchestration(reg *tools.Registry, svc *rag.Service) {
	if svc == nil {
		return
	}
	reg.Register(RAGSearch{svc: svc})
	reg.Register(RAGIndex{svc: svc})
	reg.Register(NewIndexingWriteFile(svc))
	reg.Register(NewIndexingEditFile(svc))
}

type RAGSearch struct {
	svc *rag.Service
}

func (t RAGSearch) Name() string { return "rag_search" }

func (t RAGSearch) Description() string {
	return "Search indexed knowledge bases for relevant passages. Store scope is chosen by Taiji (--local / --local-only); do not pass store_root."
}

func (t RAGSearch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Natural language search query",
			},
			"collection": map[string]any{
				"type":        "string",
				"description": "Collection name (default from config)",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "Maximum number of chunks to return (default 5)",
			},
		},
		"required": []string{"query"},
	}
}

func (t RAGSearch) Run(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	collection, _ := args["collection"].(string)
	topK := 5
	if v, ok := args["top_k"].(float64); ok && v > 0 {
		topK = int(v)
	} else if s, ok := args["top_k"].(string); ok && strings.TrimSpace(s) != "" {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return "", fmt.Errorf("top_k: %w", err)
		}
		topK = n
	}
	hits, err := t.svc.Query(ctx, rag.QueryRequest{
		Collection: collection,
		Query:      query,
		TopK:       topK,
	})
	if err != nil {
		return "", err
	}
	if len(hits) == 0 {
		return fmt.Sprintf("scope: %s\nno hits", t.svc.DescribeScope()), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "scope: %s\n", t.svc.DescribeScope())
	for i, hit := range hits {
		fmt.Fprintf(&b, "[%d] scope=%s score=%.3f source=%s\n%s\n\n", i+1, hit.Scope, hit.Score, hit.Source, hit.Text)
	}
	return strings.TrimSpace(b.String()), nil
}

type RAGIndex struct {
	svc *rag.Service
}

func (t RAGIndex) Name() string { return "rag_index" }

func (t RAGIndex) Description() string {
	return "Index files or directories into a RAG collection. Taiji writes to the active scope store automatically; do not pass store_root."
}

func (t RAGIndex) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"paths": map[string]any{
				"type":        "array",
				"description": "File or directory paths to ingest",
				"items":       map[string]any{"type": "string"},
			},
			"collection": map[string]any{
				"type":        "string",
				"description": "Collection name (default from config)",
			},
			"recursive": map[string]any{
				"type":        "boolean",
				"description": "Recursively index directories",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Optional glob filter when indexing directories (e.g. *.md)",
			},
		},
		"required": []string{"paths"},
	}
}

func (t RAGIndex) Run(ctx context.Context, args map[string]any) (string, error) {
	paths := stringSliceArg(args["paths"])
	if len(paths) == 0 {
		return "", fmt.Errorf("paths is required")
	}
	collection, _ := args["collection"].(string)
	recursive, _ := args["recursive"].(bool)
	glob, _ := args["glob"].(string)
	res, err := t.svc.Index(ctx, rag.IndexRequest{
		Collection:  collection,
		SourcePaths: paths,
		Recursive:   recursive,
		Glob:        glob,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("scope: %s (%s)\ncollection: %s\n%s (+%d chunks, %d total)",
		t.svc.DescribeScope(), res.Scope, res.Collection, res.Message, res.ChunksAdded, res.TotalChunks), nil
}

func stringSliceArg(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}
