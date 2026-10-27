package builtin

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

type GlobSearch struct{}

func (GlobSearch) Name() string { return "glob_search" }

func (GlobSearch) Description() string {
	return "Find files matching a glob pattern under a directory (e.g. *.go, **/*.md). Uses taiji-tools glob (Rust ignore + globset): respects .gitignore and standard ignore rules."
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

func (GlobSearch) Run(ctx context.Context, args map[string]any) (string, error) {
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

	matches, err := runGlobwalkBinary(ctx, root, pattern, maxResults)
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
