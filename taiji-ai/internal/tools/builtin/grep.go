package builtin

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Grep struct{}

func (Grep) Name() string { return "grep" }

func (Grep) Description() string {
	return "Search file contents with ripgrep when available, otherwise a built-in search. Use to locate symbols, strings, or definitions without knowing exact paths."
}

func (Grep) Parameters() map[string]any {
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
				"description": "Optional file filter (e.g. *.go, *.yaml)",
			},
			"case_insensitive": map[string]any{
				"type":        "boolean",
				"description": "Case-insensitive search",
			},
			"head_limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of matching lines (default 50)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (Grep) Run(ctx context.Context, args map[string]any) (string, error) {
	pattern, _ := args["pattern"].(string)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	searchPath, _ := args["path"].(string)
	glob, _ := args["glob"].(string)
	caseInsensitive, _ := args["case_insensitive"].(bool)
	headLimit := 50
	if v, ok := args["head_limit"].(float64); ok && v > 0 {
		headLimit = int(v)
	}

	root, err := explorePath(searchPath)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(root); err != nil {
		return "", err
	}

	if _, err := exec.LookPath("rg"); err == nil {
		return runRipgrep(ctx, root, pattern, glob, caseInsensitive, headLimit)
	}
	return runBuiltinGrep(root, pattern, glob, caseInsensitive, headLimit)
}

func runRipgrep(ctx context.Context, root, pattern, glob string, caseInsensitive bool, headLimit int) (string, error) {
	args := []string{
		"--line-number",
		"--no-heading",
		"--color=never",
		"--max-count", fmt.Sprintf("%d", headLimit),
	}
	if caseInsensitive {
		args = append(args, "-i")
	}
	if strings.TrimSpace(glob) != "" {
		args = append(args, "-g", strings.TrimSpace(glob))
	}
	for _, skip := range defaultSkipDirNames {
		args = append(args, "--glob", "!"+skip)
	}
	args = append(args, pattern, root)

	cmd := exec.CommandContext(ctx, "rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return fmt.Sprintf("no matches for %q under %s", pattern, root), nil
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("rg: %s", msg)
		}
		return "", err
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return fmt.Sprintf("no matches for %q under %s", pattern, root), nil
	}
	lines := strings.Split(out, "\n")
	if len(lines) > headLimit {
		lines = lines[:headLimit]
		out = strings.Join(lines, "\n") + fmt.Sprintf("\n\n(truncated at %d lines)", headLimit)
	}
	return out, nil
}

func runBuiltinGrep(root, pattern, glob string, caseInsensitive bool, headLimit int) (string, error) {
	flags := ""
	if caseInsensitive {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}

	var matches []string
	appendMatch := func(rel string, lineNum int, line string) {
		matches = append(matches, fmt.Sprintf("%s:%d:%s", rel, lineNum, strings.TrimRight(line, "\n\r")))
	}

	if !info.IsDir() {
		rel := filepath.Base(root)
		if err := grepFile(root, rel, glob, re, headLimit, appendMatch); err != nil {
			return "", err
		}
	} else {
		err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
			if err != nil || pathHasSkippedSegment(rel) {
				return nil
			}
			if len(matches) >= headLimit {
				return filepath.SkipAll
			}
			return grepFile(path, rel, glob, re, headLimit-len(matches), appendMatch)
		})
		if err != nil {
			return "", err
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("no matches for %q under %s", pattern, root), nil
	}
	sort.Strings(matches)
	out := strings.Join(matches, "\n")
	if len(matches) >= headLimit {
		out += fmt.Sprintf("\n\n(truncated at %d lines)", headLimit)
	}
	return out, nil
}

func grepFile(path, rel, glob string, re *regexp.Regexp, remaining int, appendMatch func(string, int, string)) error {
	if remaining <= 0 {
		return nil
	}
	if strings.TrimSpace(glob) != "" {
		ok, err := matchGlob(rel, glob)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return nil
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if re.FindStringIndex(line) != nil {
			appendMatch(rel, lineNum, line)
			remaining--
			if remaining <= 0 {
				break
			}
		}
	}
	return sc.Err()
}
