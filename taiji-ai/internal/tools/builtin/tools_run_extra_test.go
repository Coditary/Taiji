package builtin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func activeSessionCtx(t *testing.T) context.Context {
	t.Helper()
	root := t.TempDir()
	sess, err := session.New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	return session.WithActiveSession(context.Background(), sess)
}

func TestEditFileRunWrapper(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "w.txt")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (EditFile{}).Run(t.Context(), map[string]any{
		"path": path,
		"edits": []any{
			map[string]any{"old_text": "old", "new_text": "new"},
		},
	})
	if err != nil || !strings.Contains(out, "edited") {
		t.Fatalf("err=%v out=%q", err, out)
	}
}

func TestCreateChatToolRun(t *testing.T) {
	ctx := activeSessionCtx(t)
	out, err := (CreateChatTool{}).Run(ctx, map[string]any{
		"name":        "helper",
		"description": "demo",
		"language":    "shell",
		"script":      "echo hi",
	})
	if err != nil || !strings.Contains(out, "helper") {
		t.Fatalf("err=%v out=%q", err, out)
	}
}

func TestSetActiveBranchExplicit(t *testing.T) {
	ctx := activeSessionCtx(t)
	out, err := (SetActiveBranch{}).Run(ctx, map[string]any{"branch": "feature/x"})
	if err != nil || !strings.Contains(out, "feature/x") {
		t.Fatalf("err=%v out=%q", err, out)
	}
}

func TestTodoPlanSetGetUpdate(t *testing.T) {
	ctx := WithToolSessionID(context.Background(), "todo-sess")
	tool := TodoPlanTool{}
	setOut, err := tool.Run(ctx, map[string]any{
		"action": "set",
		"items": []any{
			map[string]any{"id": "1", "task": "alpha", "status": "pending"},
		},
	})
	if err != nil || !strings.Contains(setOut, "alpha") {
		t.Fatalf("set: err=%v out=%q", err, setOut)
	}
	getOut, err := tool.Run(ctx, map[string]any{"action": "get"})
	if err != nil || !strings.Contains(getOut, "alpha") {
		t.Fatalf("get: err=%v out=%q", err, getOut)
	}
	updOut, err := tool.Run(ctx, map[string]any{
		"action": "update",
		"items": []any{
			map[string]any{"id": "1", "status": "done"},
		},
	})
	if err != nil || !strings.Contains(updOut, "[x]") {
		t.Fatalf("update: err=%v out=%q", err, updOut)
	}
}

func TestShellEcho(t *testing.T) {
	out, err := (Shell{}).Run(t.Context(), map[string]any{"command": "echo taiji-shell-ok"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "taiji-shell-ok") {
		t.Fatalf("out=%q", out)
	}
}

func TestGrepContentSearch(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not installed")
	}
	root, ctx := useProjectWorkspace(t)
	path := filepath.Join(root, "needle.txt")
	if err := os.WriteFile(path, []byte("find the needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (Grep{}).Run(ctx, map[string]any{
		"pattern": "needle",
		"path":    root,
	})
	if err != nil || !strings.Contains(out, "needle") {
		t.Fatalf("err=%v out=%q", err, out)
	}
}

func TestWorkspaceFindWhenToolsAvailable(t *testing.T) {
	if resolveTaijiToolsBinary() == "" {
		t.Skip("taiji-tools binary not available")
	}
	root, ctx := useProjectWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "music-app.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (WorkspaceFind{}).Run(ctx, map[string]any{
		"query": "music",
		"path":  root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "music") {
		t.Fatalf("out=%q", out)
	}
}

func TestRememberInPackRun(t *testing.T) {
	root := t.TempDir()
	packRoot := filepath.Join(root, ".taiji", "packs")
	if err := os.MkdirAll(filepath.Join(packRoot, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	reloaded := false
	tool := RememberInPack{
		PackDirs: []string{packRoot},
		OnReload: func() error {
			reloaded = true
			return nil
		},
	}
	out, err := tool.Run(t.Context(), map[string]any{
		"pack_name": "demo",
		"node_name": "lore/tavern",
		"content":   "A fact about the tavern.",
	})
	if err != nil || !strings.Contains(out, "appended") {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if !reloaded {
		t.Fatal("expected OnReload")
	}
}

func TestAwaitFinishedJob(t *testing.T) {
	sid := "await-test"
	ctx := WithToolSessionID(context.Background(), sid)
	mgr := DefaultJobManager()
	id, _, err := mgr.Start(sid, "echo await-marker", "")
	if err != nil {
		t.Fatal(err)
	}
	out, err := (Await{}).Run(ctx, map[string]any{
		"job_id":         id,
		"block_until_ms": 10_000,
		"pattern":        "await-marker",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "await-marker") {
		t.Fatalf("out=%q", out)
	}
}

func TestCreateAndUpdateGoal(t *testing.T) {
	ctx := activeSessionCtx(t)
	createOut, err := (CreateGoal{}).Run(ctx, map[string]any{"objective": "ship tests"})
	if err != nil || !strings.Contains(createOut, "active") {
		t.Fatalf("create: err=%v out=%q", err, createOut)
	}
	updOut, err := (UpdateGoal{}).Run(ctx, map[string]any{"status": "complete"})
	if err != nil || !strings.Contains(updOut, "complete") {
		t.Fatalf("update: err=%v out=%q", err, updOut)
	}
}

func TestEnablePackRememberRegisters(t *testing.T) {
	reg := tools.NewRegistry()
	EnablePackRemember(reg, []string{t.TempDir()}, nil)
	EnableActorTools(reg)
	if _, ok := reg.Get("remember_in_pack"); !ok {
		t.Fatal("expected remember_in_pack registered")
	}
	if _, ok := reg.Get("update_actor_state"); !ok {
		t.Fatal("expected update_actor_state registered")
	}
}

func TestParseContentSearchArgsRejectsBadMode(t *testing.T) {
	_, err := parseContentSearchArgs(map[string]any{
		"pattern":     "x",
		"output_mode": "invalid",
	})
	if err == nil || !strings.Contains(err.Error(), "output_mode") {
		t.Fatalf("err=%v", err)
	}
}
