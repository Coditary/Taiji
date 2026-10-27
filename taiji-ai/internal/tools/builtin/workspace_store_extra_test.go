package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/question"
)

func TestWorkspaceListCheckpointsAndNoChanges(t *testing.T) {
	root := t.TempDir()
	SetWorkspaceRootsForTest(root, filepath.Join(root, "sessions"))
	mgr := DefaultWorkspace()
	sid := "list-cp"
	if _, err := mgr.ListCheckpoints(sid); err != nil {
		t.Fatal(err)
	}
	list, err := mgr.ListCheckpoints(sid)
	if err != nil || list != "no checkpoints" {
		t.Fatalf("list=%q err=%v", list, err)
	}
	_, _, err = mgr.CreateCheckpoint(sid, "empty")
	if err == nil || !strings.Contains(err.Error(), "no session file changes") {
		t.Fatalf("err=%v", err)
	}
}

func TestWorkspaceRestoreNewFileRemoval(t *testing.T) {
	root := t.TempDir()
	SetWorkspaceRootsForTest(root, filepath.Join(root, "sessions"))
	mgr := NewWorkspaceManager()
	sid := "restore-new"
	ctx := WithToolSessionID(context.Background(), sid)
	path := filepath.Join(root, "created.go")
	if err := mgr.CapturePreWrite(ctx, sid, path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr.noteModified(sid, path)
	id, _, err := mgr.CreateCheckpoint(sid, "with-new")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package main\n// edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	msg, err := mgr.Restore(ctx, sid, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "restored") {
		t.Fatalf("msg=%q", msg)
	}
}

func TestAskUserQuestionsFromResultHelpers(t *testing.T) {
	if AskUserQuestionsFromResult(nil) != nil {
		t.Fatal("nil data")
	}
	if len(AskUserQuestionsFromResult(map[string]any{"questions": []question.Info{{Question: "?"}}})) != 1 {
		t.Fatal("expected question infos")
	}
}
