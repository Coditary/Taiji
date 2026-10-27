package builtin

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Await struct{}

func (Await) Name() string { return "await" }

func (Await) Description() string {
	return "Wait for a background shell job (from bg_process) to finish or until output matches a regex. Returns recent logs and exit status."
}

func (Await) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"job_id": map[string]any{
				"type":        "string",
				"description": "Background job id from bg_process start",
			},
			"block_until_ms": map[string]any{
				"type":        "integer",
				"description": "Max wait in milliseconds (default 30000, max 600000)",
			},
			"pattern": map[string]any{
				"type":        "string",
				"description": "Optional RE2 regex; return early when job output matches",
			},
		},
		"required": []string{"job_id"},
	}
}

func (Await) Run(ctx context.Context, args map[string]any) (string, error) {
	jobID := strings.TrimSpace(stringArg(args, "job_id"))
	if jobID == "" {
		return "", fmt.Errorf("job_id is required")
	}
	waitMs := intArg(args, "block_until_ms")
	if waitMs <= 0 {
		waitMs = 30_000
	}
	if waitMs > 600_000 {
		waitMs = 600_000
	}
	var pattern *regexp.Regexp
	if raw := strings.TrimSpace(stringArg(args, "pattern")); raw != "" {
		re, err := regexp.Compile(raw)
		if err != nil {
			return "", fmt.Errorf("invalid pattern: %w", err)
		}
		pattern = re
	}
	sessionID := ToolSessionIDFrom(ctx)
	deadline := time.Now().Add(time.Duration(waitMs) * time.Millisecond)
	mgr := DefaultJobManager()
	for {
		status, logs, done, err := mgr.Status(sessionID, jobID, 80)
		if err != nil {
			return "", err
		}
		if pattern != nil && pattern.MatchString(logs) {
			return fmt.Sprintf("matched pattern (status=%s)\n%s", status, logs), nil
		}
		if done {
			return fmt.Sprintf("finished (status=%s)\n%s", status, logs), nil
		}
		if time.Now().After(deadline) {
			return fmt.Sprintf("timeout (status=%s)\n%s", status, logs), nil
		}
		if ctx != nil {
			if err := sleepContext(ctx, 250*time.Millisecond); err != nil {
				return "", err
			}
		} else {
			time.Sleep(250 * time.Millisecond)
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
