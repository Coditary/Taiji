package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/session"
)

func useProjectWorkspace(t *testing.T) (root string, ctx context.Context) {
	root = t.TempDir()
	sessions := filepath.Join(root, ".taiji", "sessions")
	t.Cleanup(func() { SetWorkspaceRootsForTest(".", "") })
	SetWorkspaceRootsForTest(root, sessions)
	ctx = context.Background()
	return root, ctx
}

func TestReadFileAndListDir(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	readOut, err := (ReadFile{}).Run(ctx, map[string]any{"path": path})
	if err != nil || !strings.Contains(readOut, "alpha") {
		t.Fatalf("read: err=%v out=%q", err, readOut)
	}
	listOut, err := (ListDir{}).Run(ctx, map[string]any{"path": root})
	if err != nil || !strings.Contains(listOut, "notes.txt") {
		t.Fatalf("list: err=%v out=%q", err, listOut)
	}
}

func TestGrepAndGlobInWorkspace(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "findme.go"), []byte("package main\n// needle here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	grepOut, err := (Grep{}).Run(ctx, map[string]any{
		"pattern": "needle",
		"path":    root,
	})
	if err != nil || !strings.Contains(grepOut, "needle") {
		t.Fatalf("grep: err=%v out=%q", err, grepOut)
	}
	if resolveTaijiToolsBinary() != "" {
		globOut, err := (GlobSearch{}).Run(ctx, map[string]any{
			"pattern": "*.go",
			"path":    root,
		})
		if err != nil || !strings.Contains(globOut, "findme.go") {
			t.Fatalf("glob: err=%v out=%q", err, globOut)
		}
	}
}

func TestDeleteFileTool(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	path := filepath.Join(root, "gone.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (DeleteFile{}).Run(ctx, map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "deleted") && !strings.Contains(out, "Deleted") {
		t.Fatalf("out=%q", out)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file should be removed")
	}
}

func TestSessionBoundTools(t *testing.T) {
	root, _ := useProjectWorkspace(t)
	sess, err := session.New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	ctx := session.WithActiveSession(context.Background(), sess)

	modeOut, err := (SwitchAgentMode{}).Run(ctx, map[string]any{"mode": "plan"})
	if err != nil || !strings.Contains(modeOut, "plan") {
		t.Fatalf("mode: err=%v out=%q", err, modeOut)
	}
	stateOut, err := (UpdateActorState{}).Run(ctx, map[string]any{"key": "hp", "value": "100"})
	if err != nil || !strings.Contains(stateOut, "hp") {
		t.Fatalf("state: err=%v out=%q", err, stateOut)
	}
	if sess.State.Settings.ActorState["hp"] != "100" {
		t.Fatalf("actor state=%v", sess.State.Settings.ActorState)
	}
}

func TestToolSessionIDContext(t *testing.T) {
	ctx := WithToolSessionID(context.Background(), "sess-42")
	if ToolSessionIDFrom(ctx) != "sess-42" {
		t.Fatalf("id=%q", ToolSessionIDFrom(ctx))
	}
}
