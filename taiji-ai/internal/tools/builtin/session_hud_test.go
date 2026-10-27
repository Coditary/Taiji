package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionHUDTodoAndCheckpoint(t *testing.T) {
	root := t.TempDir()
	SetWorkspaceRootsForTest(root, filepath.Join(root, "sessions"))
	sid := "hud-sess"
	InvalidateSessionHUDCache(sid)

	_, err := (TodoPlanTool{}).Run(WithToolSessionID(context.Background(), sid), map[string]any{
		"action": "set",
		"items": []any{
			map[string]any{"id": "1", "task": "Tests schreiben", "status": "in_progress"},
			map[string]any{"id": "2", "task": "Docs", "status": "pending"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	mgr := DefaultWorkspace()
	path := filepath.Join(root, "x.go")
	if err := os.WriteFile(path, []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mgr.CapturePreWrite(WithToolSessionID(context.Background(), sid), sid, path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package x\n// change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr.noteModified(sid, path)
	_, _, err = mgr.CreateCheckpoint(sid, "pre-refactor")
	if err != nil {
		t.Fatal(err)
	}

	hud := GetSessionHUD(sid)
	if hud.TodoSummary == "" || hud.ActiveJobCount != 0 {
		t.Fatalf("todo: %#v", hud)
	}
	if hud.LatestCheckpoint == "" {
		t.Fatal("expected checkpoint in hud")
	}
	line := FormatSessionHUDLine(hud)
	if line == "" {
		t.Fatal("expected footer line")
	}

	sidebar := FormatTodosSidebar(sid)
	if sidebar == "" || !strings.Contains(sidebar, "[>] Tests schreiben") || !strings.Contains(sidebar, "[ ] Docs") {
		t.Fatalf("sidebar todos: %q", sidebar)
	}
}
