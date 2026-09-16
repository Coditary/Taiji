package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ListDir struct{}

func (ListDir) Name() string { return "list_dir" }

func (ListDir) Description() string {
	return "List files and directories in a path. Use this to explore the project before reading files."
}

func (ListDir) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Directory path (default: current directory)",
			},
			"recursive": map[string]any{
				"type":        "boolean",
				"description": "List subdirectories recursively (default false)",
			},
			"max_depth": map[string]any{
				"type":        "integer",
				"description": "Maximum depth when recursive (default 2)",
			},
		},
	}
}

func (ListDir) Run(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	recursive, _ := args["recursive"].(bool)
	maxDepth := 2
	if v, ok := args["max_depth"].(float64); ok && v > 0 {
		maxDepth = int(v)
	}

	abs, err := explorePath(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", abs)
	}

	if !recursive {
		return formatDirListing(abs, abs, false)
	}
	return formatRecursiveListing(abs, maxDepth)
}

func formatDirListing(root, dir string, prefixRel bool) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	var lines []string
	for _, ent := range entries {
		if isSkippedPath(ent.Name()) {
			continue
		}
		full := filepath.Join(dir, ent.Name())
		rel := ent.Name()
		if prefixRel {
			rel, _ = filepath.Rel(root, full)
		}
		if ent.IsDir() {
			lines = append(lines, fmt.Sprintf("%s/ (dir)", rel))
			continue
		}
		info, err := ent.Info()
		if err != nil {
			lines = append(lines, fmt.Sprintf("%s (file)", rel))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s (file, %d bytes)", rel, info.Size()))
	}
	if len(lines) == 0 {
		return fmt.Sprintf("%s\n(empty)", dir), nil
	}
	return strings.Join(lines, "\n"), nil
}

func formatRecursiveListing(root string, maxDepth int) (string, error) {
	var lines []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if pathHasSkippedSegment(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		depth := strings.Count(filepath.ToSlash(rel), "/") + 1
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			lines = append(lines, fmt.Sprintf("%s/ (dir)", rel))
			return nil
		}
		info, err := d.Info()
		if err != nil {
			lines = append(lines, fmt.Sprintf("%s (file)", rel))
			return nil
		}
		lines = append(lines, fmt.Sprintf("%s (file, %d bytes)", rel, info.Size()))
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return fmt.Sprintf("%s\n(empty)", root), nil
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}
