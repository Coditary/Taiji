package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Shell struct{}

func (Shell) Name() string { return "shell" }

func (Shell) Description() string {
	return "Run a shell command and return combined stdout/stderr."
}

func (Shell) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to execute",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory (optional)",
			},
		},
		"required": []string{"command"},
	}
}

func (Shell) Run(ctx context.Context, args map[string]any) (string, error) {
	command, _ := args["command"].(string)
	cwd, _ := args["cwd"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("command is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "sh", "-c", command)
	if strings.TrimSpace(cwd) != "" {
		cmd.Dir = cwd
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		if out == "" {
			return "", err
		}
		return out + "\n(error: " + err.Error() + ")", nil
	}
	return out, nil
}
