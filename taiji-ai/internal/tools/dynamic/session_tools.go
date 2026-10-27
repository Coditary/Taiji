package dynamic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

const chatToolTimeout = 20 * time.Second

// SessionChatTool implements a single session-defined script tool.
type SessionChatTool struct {
	Def     session.ChatTool
	WorkDir string
}

func (t SessionChatTool) Name() string { return t.Def.Name }

func (t SessionChatTool) Description() string { return t.Def.Description }

func (t SessionChatTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string", "description": "Optional text passed to the tool"},
		},
	}
}

func (t SessionChatTool) Run(ctx context.Context, args map[string]any) (string, error) {
	input := ""
	if v, ok := args["input"].(string); ok {
		input = v
	}
	workDir := strings.TrimSpace(t.WorkDir)
	if workDir == "" {
		workDir = "."
	}
	dir := filepath.Join(workDir, ".taiji", "chat-tools", t.Def.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ctxRun, cancel := context.WithTimeout(ctx, chatToolTimeout)
	defer cancel()
	env := append(os.Environ(), "TAIJI_TOOL_INPUT="+input)
	var cmd *exec.Cmd
	switch t.Def.Language {
	case "python":
		py := filepath.Join(dir, "tool.py")
		if err := os.WriteFile(py, []byte(t.Def.Script), 0o600); err != nil {
			return "", err
		}
		cmd = exec.CommandContext(ctxRun, "python3", py)
	case "shell":
		sh := filepath.Join(dir, "tool.sh")
		if err := os.WriteFile(sh, []byte(t.Def.Script), 0o700); err != nil {
			return "", err
		}
		cmd = exec.CommandContext(ctxRun, "/bin/sh", sh)
	default:
		return "", fmt.Errorf("unsupported language %q", t.Def.Language)
	}
	cmd.Dir = dir
	cmd.Env = env
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(raw)))
	}
	return strings.TrimSpace(string(raw)), nil
}

// ExtendRegistry returns a registry that includes session chat tools.
func ExtendRegistry(base *tools.Registry, s *session.Session, workDir string) *tools.Registry {
	if base == nil {
		return tools.NewRegistry()
	}
	if s == nil || len(s.State.Settings.ChatTools) == 0 {
		return base
	}
	out := tools.NewRegistry()
	for _, t := range base.List() {
		out.Register(t)
	}
	for _, def := range s.State.Settings.ChatTools {
		out.Register(SessionChatTool{Def: def, WorkDir: workDir})
	}
	return out
}
