package builtin

import (
	"context"
	"fmt"
	"strings"
)

type BgProcess struct{}

func (BgProcess) Name() string { return "bg_process" }

func (BgProcess) Description() string {
	return "Manage background shell jobs (servers, watchers). Actions: start (returns job_id), logs (tail output), kill. Jobs are scoped to the current chat session."
}

func (BgProcess) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "start | logs | kill",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command (required for start)",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory (optional, for start)",
			},
			"job_id": map[string]any{
				"type":        "string",
				"description": "Job id (required for logs and kill)",
			},
			"lines": map[string]any{
				"type":        "integer",
				"description": "Lines of output for logs (default 50)",
			},
		},
		"required": []string{"action"},
	}
}

func (BgProcess) Run(ctx context.Context, args map[string]any) (string, error) {
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	sessionID := ToolSessionIDFrom(ctx)
	mgr := DefaultJobManager()

	switch action {
	case "start":
		command := strings.TrimSpace(stringArg(args, "command"))
		cwd := strings.TrimSpace(stringArg(args, "cwd"))
		id, pid, err := mgr.Start(sessionID, command, cwd)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("started job_id=%s pid=%d (session=%s)", id, pid, sessionOrDefault(sessionID)), nil
	case "logs":
		jobID := strings.TrimSpace(stringArg(args, "job_id"))
		if jobID == "" {
			return "", fmt.Errorf("job_id is required for logs")
		}
		return mgr.Logs(sessionID, jobID, intArg(args, "lines"))
	case "kill":
		jobID := strings.TrimSpace(stringArg(args, "job_id"))
		if jobID == "" {
			return "", fmt.Errorf("job_id is required for kill")
		}
		return mgr.Kill(sessionID, jobID)
	default:
		return "", fmt.Errorf("unknown action %q (use start, logs, or kill)", action)
	}
}

func sessionOrDefault(id string) string {
	if strings.TrimSpace(id) == "" {
		return "default"
	}
	return id
}
