package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type GitInspect struct{}

func (GitInspect) Name() string { return "git_inspect" }

func (GitInspect) Description() string {
	return "Read-only git introspection: status, log, blame, or show commit (no shell required)."
}

func (GitInspect) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"status", "log", "blame", "show"},
				"description": "Git operation",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File path (log, blame, show)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Log entry limit (default 10, max 30)",
			},
			"start_line": map[string]any{"type": "integer"},
			"end_line":   map[string]any{"type": "integer"},
			"commit": map[string]any{
				"type":        "string",
				"description": "Commit hash or ref for show",
			},
		},
		"required": []string{"action"},
	}
}

func (GitInspect) Run(ctx context.Context, args map[string]any) (string, error) {
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	switch action {
	case "status":
		return gitInspectStatus()
	case "log":
		return gitInspectLog(args)
	case "blame":
		return gitInspectBlame(args)
	case "show":
		return gitInspectShow(args)
	default:
		return "", fmt.Errorf("action must be status, log, blame, or show")
	}
}

func gitInspectStatus() (string, error) {
	root := workspaceProjectRoot()
	if root == "" {
		root = "."
	}
	if resolveTaijiToolsBinary() != "" {
		out, err := runTaijiTools(context.Background(), "git-status", "--root", root)
		if err == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out), nil
		}
	}
	branch, err := runGitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("not a git repository or git unavailable: %w", err)
	}
	short, err := runGitOutput("status", "-sb")
	if err != nil {
		return "", err
	}
	if short == "" {
		short = "## " + branch + "\n(clean)"
	}
	return "branch: " + strings.TrimSpace(branch) + "\n" + short, nil
}

func gitInspectLog(args map[string]any) (string, error) {
	limit := intArg(args, "limit")
	if limit <= 0 {
		limit = 10
	}
	if limit > 30 {
		limit = 30
	}
	root := workspaceProjectRoot()
	if root == "" {
		root = "."
	}
	if resolveTaijiToolsBinary() != "" {
		out, err := runTaijiTools(context.Background(), "git-log", "--root", root, "--limit", fmt.Sprintf("%d", limit))
		if err == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out), nil
		}
	}
	path := stringArg(args, "path")
	gitArgs := []string{
		"log",
		fmt.Sprintf("-n%d", limit),
		"--date=short",
		"--pretty=format:%h %ad | %an: %s",
	}
	if path != "" {
		gitArgs = append(gitArgs, "--", path)
	}
	out, err := runGitOutput(gitArgs...)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "(no commits)", nil
	}
	return out, nil
}

func gitInspectBlame(args map[string]any) (string, error) {
	path := stringArg(args, "path")
	if path == "" {
		return "", fmt.Errorf("path is required for blame")
	}
	start := intArg(args, "start_line")
	end := intArg(args, "end_line")
	if start <= 0 || end <= 0 || end < start {
		return "", fmt.Errorf("start_line and end_line are required for blame")
	}
	lineRange := fmt.Sprintf("%d,%d", start, end)
	return runGitOutput("blame", "-L", lineRange, "--date=short", "-w", path)
}

func gitInspectShow(args map[string]any) (string, error) {
	commit := stringArg(args, "commit")
	if commit == "" {
		return "", fmt.Errorf("commit is required for show")
	}
	path := stringArg(args, "path")
	gitArgs := []string{"show", "--stat", "--patch", "--no-color", commit}
	if path != "" {
		gitArgs = append(gitArgs, "--", path)
	}
	out, err := runGitOutput(gitArgs...)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if len(out) > 8000 {
		out = out[:8000] + "\n[... truncated ...]"
	}
	return out, nil
}

func runGitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workspaceProjectRoot()
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), text)
	}
	return text, nil
}
