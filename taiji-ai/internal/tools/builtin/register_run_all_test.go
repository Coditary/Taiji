package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func TestRegisteredToolsExtendedSmoke(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	sess, err := session.New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	ctx = session.WithActiveSession(ctx, sess)
	sidCtx := WithToolSessionID(ctx, sess.State.ID)

	reg := tools.NewRegistry()
	Register(reg)

	txt := filepath.Join(root, "note.txt")
	goFile := filepath.Join(root, "main.go")
	nb := filepath.Join(root, "nb.ipynb")
	_ = os.WriteFile(txt, []byte("needle line\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "del-me.txt"), []byte("x"), 0o644)
	_ = os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0o644)
	_ = os.WriteFile(nb, []byte(`{"nbformat":4,"nbformat_minor":5,"metadata":{},"cells":[]}`), 0o644)

	cases := []struct {
		name string
		args map[string]any
		ctx  context.Context
	}{
		{"write_file", map[string]any{"path": filepath.Join(root, "out.txt"), "content": "data"}, ctx},
		{"manage_file", map[string]any{"action": "delete", "path": filepath.Join(root, "del-me.txt")}, sidCtx},
		{"file_outline", map[string]any{"path": goFile}, ctx},
		{"go_symbol_intel", map[string]any{"path": goFile, "symbol": "main"}, ctx},
		{"workspace_find", map[string]any{"query": "needle", "path": root}, ctx},
		{"search_conversations", map[string]any{"query": "needle"}, ctx},
		{"edit_notebook", map[string]any{
			"target_notebook": nb, "cell_idx": 0, "is_new_cell": true,
			"cell_language": "python", "old_string": "", "new_string": "1+1",
		}, ctx},
		{"create_goal", map[string]any{"objective": "cover tests"}, ctx},
		{"update_goal", map[string]any{"status": "active"}, ctx},
		{"bg_process", map[string]any{"action": "list"}, ctx},
		{"await", map[string]any{"block_until_ms": 1}, sidCtx},
		{"web_search", map[string]any{"query": "taiji"}, ctx},
		{"shell", map[string]any{"command": "echo ok"}, ctx},
	}

	for _, tc := range cases {
		tool, ok := reg.Get(tc.name)
		if !ok {
			t.Fatalf("missing %s", tc.name)
		}
		runCtx := tc.ctx
		if runCtx == nil {
			runCtx = ctx
		}
		_, err := tool.Run(runCtx, tc.args)
		if err != nil && !extendedSmokeOK(tc.name, err) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func extendedSmokeOK(name string, err error) bool {
	msg := err.Error()
	switch name {
	case "go_symbol_intel", "read_lints", "web_fetch", "web_search", "run_tests", "git_inspect",
		"workspace_find", "glob_search", "grep", "search_conversations", "bg_process", "await":
		return true
	default:
		return strings.Contains(msg, "not found") ||
			strings.Contains(msg, "skip") ||
			strings.Contains(msg, "no active") ||
			strings.Contains(msg, "unavailable")
	}
}
