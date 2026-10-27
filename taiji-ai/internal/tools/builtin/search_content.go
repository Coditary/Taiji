package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type contentSearchArgs struct {
	pattern         string
	root            string
	glob            string
	caseInsensitive bool
	headLimit       int
	outputMode      string
	contextLines    int
	fileType        string
}

func contentSearchToolParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Search pattern (regex by default)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory to search (default: current directory)",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Optional file filter (e.g. *.go, **/*.yaml)",
			},
			"case_insensitive": map[string]any{
				"type":        "boolean",
				"description": "Case-insensitive search",
			},
			"head_limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of matching lines or paths (default 50)",
			},
			"output_mode": map[string]any{
				"type":        "string",
				"enum":        []string{"content", "files_with_matches", "count"},
				"description": "content: matching lines; files_with_matches: paths only; count: match counts per file",
			},
			"context_lines": map[string]any{
				"type":        "integer",
				"description": "Lines of context before and after each match (requires rg)",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "File type filter when using rg (e.g. go, rust, ts)",
			},
		},
		"required": []string{"pattern"},
	}
}

func parseContentSearchArgs(args map[string]any) (contentSearchArgs, error) {
	pattern, _ := args["pattern"].(string)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return contentSearchArgs{}, fmt.Errorf("pattern is required")
	}
	searchPath, _ := args["path"].(string)
	glob, _ := args["glob"].(string)
	caseInsensitive, _ := args["case_insensitive"].(bool)
	headLimit := 50
	if v, ok := args["head_limit"].(float64); ok && v > 0 {
		headLimit = int(v)
	}
	outputMode, _ := args["output_mode"].(string)
	outputMode = strings.TrimSpace(strings.ToLower(outputMode))
	if outputMode == "" {
		outputMode = "content"
	}
	switch outputMode {
	case "content", "files_with_matches", "count":
	default:
		return contentSearchArgs{}, fmt.Errorf("output_mode must be content, files_with_matches, or count")
	}
	contextLines := 0
	if v, ok := args["context_lines"].(float64); ok && v > 0 {
		contextLines = int(v)
	}
	fileType, _ := args["type"].(string)
	fileType = strings.TrimSpace(fileType)

	root, err := explorePath(searchPath)
	if err != nil {
		return contentSearchArgs{}, err
	}
	if _, err := os.Stat(root); err != nil {
		return contentSearchArgs{}, err
	}

	return contentSearchArgs{
		pattern:         pattern,
		root:            root,
		glob:            glob,
		caseInsensitive: caseInsensitive,
		headLimit:       headLimit,
		outputMode:      outputMode,
		contextLines:    contextLines,
		fileType:        fileType,
	}, nil
}

// RunContentSearch searches file contents using ripgrep only.
func RunContentSearch(ctx context.Context, args map[string]any) (string, error) {
	parsed, err := parseContentSearchArgs(args)
	if err != nil {
		return "", err
	}
	rg, err := requireRipgrep()
	if err != nil {
		return "", err
	}
	return runRipgrep(ctx, rg, parsed)
}

func runRipgrep(ctx context.Context, rgBin string, parsed contentSearchArgs) (string, error) {
	args := []string{"--color=never"}
	switch parsed.outputMode {
	case "files_with_matches":
		args = append(args, "--files-with-matches")
	case "count":
		args = append(args, "--count")
	default:
		args = append(args, "--line-number", "--no-heading")
		if parsed.contextLines > 0 {
			args = append(args, "-C", fmt.Sprintf("%d", parsed.contextLines))
		} else {
			args = append(args, "--max-count", fmt.Sprintf("%d", parsed.headLimit))
		}
	}
	if parsed.caseInsensitive {
		args = append(args, "-i")
	}
	if strings.TrimSpace(parsed.glob) != "" {
		args = append(args, "-g", strings.TrimSpace(parsed.glob))
	}
	if parsed.fileType != "" {
		args = append(args, "-t", parsed.fileType)
	}
	for _, skip := range defaultSkipDirNames {
		args = append(args, "--glob", "!"+skip)
	}
	args = append(args, parsed.pattern, parsed.root)

	cmd := exec.CommandContext(ctx, rgBin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return noMatchesMessage(parsed), nil
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("rg: %s", msg)
		}
		return "", err
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return noMatchesMessage(parsed), nil
	}
	lines := strings.Split(out, "\n")
	if len(lines) > parsed.headLimit {
		lines = lines[:parsed.headLimit]
		out = strings.Join(lines, "\n") + fmt.Sprintf("\n\n(truncated at %d lines)", parsed.headLimit)
	}
	return out, nil
}

func noMatchesMessage(parsed contentSearchArgs) string {
	return fmt.Sprintf("no matches for %q under %s", parsed.pattern, parsed.root)
}

