package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func TestRegisteredToolsSmokeRun(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	sess, err := session.New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	ctx = session.WithActiveSession(ctx, sess)
	reg := tools.NewRegistry()
	Register(reg)

	// Prepare fixtures used by multiple tools.
	txt := filepath.Join(root, "note.txt")
	if err := os.WriteFile(txt, []byte("needle line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args map[string]any
	}{
		{"list_dir", map[string]any{"path": root}},
		{"read_file", map[string]any{"path": txt}},
		{"file_outline", map[string]any{"path": txt}},
		{"glob_search", map[string]any{"glob_pattern": "*.txt", "target_directory": root}},
		{"grep", map[string]any{"pattern": "needle", "path": root}},
		{"workspace_checkpoint", map[string]any{"action": "list"}},
		{"todo_plan", map[string]any{"action": "get"}},
		{"switch_agent_mode", map[string]any{"mode": "agent"}},
		{"set_active_branch", map[string]any{"branch": "main"}},
		{"git_inspect", map[string]any{"action": "status"}},
		{"run_tests", map[string]any{"command": "echo smoke", "timeout": "5s"}},
		{"read_lints", map[string]any{"paths": []any{txt}}},
		{"delete_file", map[string]any{"path": txt}},
	}

	for _, tc := range cases {
		name, args := tc.name, tc.args
		tool, ok := reg.Get(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if tool.Description() == "" || tool.Parameters() == nil {
			t.Fatalf("%s metadata", name)
		}
		_, err := tool.Run(ctx, args)
		if err != nil && !acceptableSmokeError(name, err) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func acceptableSmokeError(name string, err error) bool {
	msg := err.Error()
	switch name {
	case "web_fetch", "run_tests", "git_inspect", "read_lints", "glob_search", "grep":
		return true
	default:
		return strings.Contains(msg, "not found") || strings.Contains(msg, "skip")
	}
}
