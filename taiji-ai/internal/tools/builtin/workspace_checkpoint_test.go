package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceCheckpointCreateDiffRestore(t *testing.T) {
	root := t.TempDir()
	sessions := filepath.Join(root, "sessions")
	SetWorkspaceRootsForTest(root, sessions)
	mgr := NewWorkspaceManager()
	ctx := WithToolSessionID(context.Background(), "sess-a")

	path := filepath.Join(root, "demo.txt")
	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mgr.CapturePreWrite(ctx, "sess-a", path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr.noteModified("sess-a", path)

	id, _, err := mgr.CreateCheckpoint("sess-a", "before-v3")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("v3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diff, err := mgr.Diff("sess-a", id, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "-v2") || !strings.Contains(diff, "+v3") {
		t.Fatalf("diff=%q", diff)
	}
	msg, err := mgr.Restore(ctx, "sess-a", id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "restored") {
		t.Fatalf("msg=%q", msg)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "v2\n" {
		t.Fatalf("file=%q", raw)
	}
}

func TestTodoPlanSetAndUpdate(t *testing.T) {
	SetWorkspaceRootsForTest(t.TempDir(), filepath.Join(t.TempDir(), "sessions"))
	ctx := WithToolSessionID(context.Background(), "todo-sess")
	out, err := (TodoPlanTool{}).Run(ctx, map[string]any{
		"action": "set",
		"items": []any{
			map[string]any{"id": "1", "task": "build parser", "status": "pending"},
			map[string]any{"id": "2", "task": "write tests", "status": "pending"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[ ] 1.") {
		t.Fatalf("set out=%q", out)
	}
	out, err = (TodoPlanTool{}).Run(ctx, map[string]any{
		"action": "update",
		"items": []any{
			map[string]any{"id": "1", "status": "in_progress"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[>] 1.") {
		t.Fatalf("update out=%q", out)
	}
	rem := DefaultTodoManager().OpenReminder("todo-sess")
	if rem == "" || !strings.Contains(rem, "build parser") {
		t.Fatalf("reminder=%q", rem)
	}
}
