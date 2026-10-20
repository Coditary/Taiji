package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultShellTimeout = 30 * time.Second

func runShellAction(ctx context.Context, workDir string, spec ActionSpec, input ShellInput) (Result, error) {
	timeout := defaultShellTimeout
	if spec.Timeout > 0 {
		timeout = time.Duration(spec.Timeout) * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch {
	case strings.TrimSpace(spec.Bash) != "":
		script := expandTemplate(strings.TrimSpace(spec.Bash), input)
		cmd = exec.CommandContext(runCtx, "bash", "-lc", script)
	case strings.TrimSpace(spec.Command) != "":
		cmd = exec.CommandContext(runCtx, "bash", "-lc", expandTemplate(strings.TrimSpace(spec.Command), input))
	default:
		return Result{}, fmt.Errorf("shell action requires bash or command")
	}
	cmd.Dir = firstNonEmpty(workDir, ".")
	cmd.Env = os.Environ()

	payload, err := json.Marshal(input)
	if err != nil {
		return Result{}, err
	}
	cmd.Stdin = bytes.NewReader(payload)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = "hook blocked action"
		}
		return Result{Block: true, Message: msg}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("hook command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	result, err := decodeShellResult(stdout.String())
	if err != nil {
		return Result{}, fmt.Errorf("parse hook stdout: %w", err)
	}
	if msg := strings.TrimSpace(stderr.String()); msg != "" && result.AdditionalContext == "" {
		result.AdditionalContext = msg
	}
	return result, nil
}

func expandTemplate(text string, input ShellInput) string {
	if !strings.Contains(text, "{{") {
		return text
	}
	files := ""
	if input.File != nil && len(input.File.Files) > 0 {
		files = strings.Join(input.File.Files, " ")
	}
	replacer := strings.NewReplacer(
		"{{files}}", files,
		"{{tool}}", toolName(input),
		"{{work_dir}}", input.Env.WorkDir,
		"{{session_id}}", input.Env.SessionID,
	)
	return replacer.Replace(text)
}

func toolName(input ShellInput) string {
	if input.Tool != nil {
		return input.Tool.Tool
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ExtractFilesFromArgs reads file paths from common tool argument JSON.
func ExtractFilesFromArgs(toolName, argsJSON string) []string {
	toolName = strings.TrimSpace(toolName)
	if argsJSON == "" {
		return nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil
	}
	switch toolName {
	case "write_file", "edit_file", "read_file", "manage_file":
		if raw, ok := args["files"].([]any); ok && len(raw) > 0 {
			var paths []string
			for _, item := range raw {
				if m, ok := item.(map[string]any); ok {
					if p, ok := m["path"].(string); ok && strings.TrimSpace(p) != "" {
						paths = append(paths, strings.TrimSpace(p))
					}
				}
			}
			if len(paths) > 0 {
				return paths
			}
		}
		if path, ok := args["path"].(string); ok && path != "" {
			out := []string{path}
			if dest, ok := args["dest"].(string); ok && strings.TrimSpace(dest) != "" {
				out = append(out, strings.TrimSpace(dest))
			}
			return out
		}
	case "batch_edit":
		if raw, ok := args["files"].([]any); ok {
			var paths []string
			for _, item := range raw {
				if m, ok := item.(map[string]any); ok {
					if p, ok := m["path"].(string); ok && strings.TrimSpace(p) != "" {
						paths = append(paths, strings.TrimSpace(p))
					}
				}
			}
			return paths
		}
	}
	return nil
}

func absPath(workDir, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	if workDir == "" {
		return path
	}
	return filepath.Join(workDir, path)
}
