package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/coditary/taiji/internal/session"
)

type SetActiveBranch struct{}

func (SetActiveBranch) Name() string { return "set_active_branch" }

func (SetActiveBranch) Description() string {
	return "Set the active git branch metadata for this chat (shown in UI / used for context). Pass branch name or omit to use current HEAD."
}

func (SetActiveBranch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"branch": map[string]any{
				"type":        "string",
				"description": "Branch name (optional; defaults to git rev-parse --abbrev-ref HEAD in project root)",
			},
		},
	}
}

func (SetActiveBranch) Run(ctx context.Context, args map[string]any) (string, error) {
	sess, ok := session.ActiveSessionFrom(ctx)
	if !ok {
		return "", fmt.Errorf("no active session")
	}
	branch := strings.TrimSpace(stringArg(args, "branch"))
	if branch == "" {
		root := workspaceProjectRoot()
		if root == "" {
			root = "."
		}
		out, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return "", fmt.Errorf("could not detect git branch: %w", err)
		}
		branch = strings.TrimSpace(string(out))
	}
	if branch == "" {
		return "", fmt.Errorf("branch is empty")
	}
	sess.State.Settings.ActiveGitBranch = branch
	if err := sess.Save(); err != nil {
		return "", err
	}
	return "active branch set to " + branch, nil
}
