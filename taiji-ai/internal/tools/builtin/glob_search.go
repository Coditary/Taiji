package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type GlobSearch struct{}

func (GlobSearch) Name() string { return "glob_search" }

func (GlobSearch) Description() string {
	return "Find files matching a glob pattern under a directory (e.g. *.go, **/*.md). Skips node_modules, .git, and similar."
}

func (GlobSearch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern (e.g. *.go, **/*.yaml)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Root directory to search (default: current directory)",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of paths to return (default 100)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (GlobSearch) Run(_ context.Context, args map[string]any) (string, error) {
	pattern, _ := args["pattern"].(string)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	rootPath, _ := args["path"].(string)
	maxResults := 100
	if v, ok := args["max_results"].(float64); ok && v > 0 {
		maxResults = int(v)
	}

	root, err := explorePath(rootPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", root)
	}

	matches, err := findGlobMatches(root, pattern, maxResults)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return fmt.Sprintf("no files matching %q under %s", pattern, root), nil
	}
	sort.Strings(matches)
	out := strings.Join(matches, "\n")
	if len(matches) >= maxResults {
		out += fmt.Sprintf("\n\n(truncated at %d results)", maxResults)
	}
	return out, nil
}

func findGlobMatches(root, pattern string, maxResults int) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && isSkippedPath(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if pathHasSkippedSegment(rel) {
			return nil
		}
		ok, err := matchGlob(rel, pattern)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		matches = append(matches, rel)
		if len(matches) >= maxResults {
			return filepath.SkipAll
		}
		return nil
	})
	return matches, err
}
