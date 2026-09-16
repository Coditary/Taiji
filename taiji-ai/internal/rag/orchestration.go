package rag

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

var autoIndexSkipDirParts = []string{
	".wuji",
	".taiji",
	".git",
	".cache",
	"node_modules",
	"vendor",
	"__pycache__",
}

var autoIndexExtensions = map[string]struct{}{
	".md": {}, ".txt": {}, ".rst": {}, ".adoc": {},
	".go": {}, ".py": {}, ".rs": {}, ".js": {}, ".ts": {},
	".jsx": {}, ".tsx": {}, ".yaml": {}, ".yml": {}, ".json": {},
	".html": {}, ".htm": {}, ".css": {}, ".scss": {}, ".sql": {},
	".sh": {}, ".bash": {}, ".zsh": {}, ".toml": {}, ".xml": {},
	".csv": {}, ".proto": {}, ".vue": {}, ".svelte": {},
}

// SystemPromptAppendix tells the agent how Taiji handles RAG scope automatically.
func SystemPromptAppendix(s *Service) string {
	if s == nil {
		return ""
	}
	writeRoot := s.Scope.WriteRoot()
	return strings.TrimSpace(fmt.Sprintf(`## Knowledge base (RAG)

Taiji chooses where indexed content is stored. Do not set store_root or run the wuji CLI for indexing.

Active scope: %s
Write store: %s
Default collection: %s

Tools:
- rag_search — search indexed knowledge before reading many large files
- rag_index — ingest files or directories when docs change or search misses content

Indexing via rag_index (and after write_file for text files) writes to the active scope automatically. Prefer rag_search for project-specific questions; index new or updated documentation so later turns can retrieve it.`,
		s.DescribeScope(), writeRoot, s.DefaultCollection))
}

// ShouldAutoIndexPath reports whether a written file should be indexed automatically.
func ShouldAutoIndexPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	for _, part := range autoIndexSkipDirParts {
		if strings.Contains(clean, "/"+part+"/") || strings.HasPrefix(clean, part+"/") {
			return false
		}
	}
	ext := strings.ToLower(filepath.Ext(clean))
	if ext == "" {
		return false
	}
	_, ok := autoIndexExtensions[ext]
	return ok
}

// AutoIndexFile indexes one file into the default collection at the active write scope.
func AutoIndexFile(ctx context.Context, s *Service, path string) (*IndexResult, error) {
	if s == nil {
		return nil, fmt.Errorf("rag service is not configured")
	}
	if !ShouldAutoIndexPath(path) {
		return nil, nil
	}
	return s.Index(ctx, IndexRequest{SourcePaths: []string{path}})
}

// FormatAutoIndexNote returns a short observation suffix for tool output.
func FormatAutoIndexNote(res *IndexResult, err error) string {
	if err != nil {
		return fmt.Sprintf("[rag] auto-index skipped: %v", err)
	}
	if res == nil {
		return ""
	}
	return fmt.Sprintf("[rag] auto-indexed into %q (%s, +%d chunks)",
		res.Collection, res.Scope, res.ChunksAdded)
}
