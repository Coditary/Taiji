package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/session"
)

func TestEditNotebookCreatesFirstCell(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	nbPath := filepath.Join(root, "demo.ipynb")
	if err := os.WriteFile(nbPath, []byte(`{"nbformat":4,"nbformat_minor":5,"metadata":{},"cells":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (EditNotebook{}).Run(ctx, map[string]any{
		"target_notebook": nbPath,
		"cell_idx":        0,
		"is_new_cell":     true,
		"cell_language":   "python",
		"old_string":      "",
		"new_string":      "print('hi')",
	})
	if err != nil || !strings.Contains(out, "updated notebook") {
		t.Fatalf("err=%v out=%q", err, out)
	}
	raw, err := os.ReadFile(nbPath)
	if err != nil || !strings.Contains(string(raw), "print('hi')") {
		t.Fatalf("file=%q err=%v", raw, err)
	}
}

func TestBgProcessStartLogsKill(t *testing.T) {
	ctx := session.WithActiveSession(context.Background(), mustSession(t))
	startOut, err := (BgProcess{}).Run(ctx, map[string]any{
		"action":  "start",
		"command": "echo bg-process-marker",
	})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Fields(startOut)
	var id string
	for _, p := range parts {
		if strings.HasPrefix(p, "job_id=") {
			id = strings.TrimPrefix(p, "job_id=")
			break
		}
	}
	if id == "" {
		t.Fatalf("parse job id from %q", startOut)
	}
	logs, err := (Await{}).Run(ctx, map[string]any{
		"job_id":         id,
		"block_until_ms": 5000,
		"pattern":        "bg-process-marker",
	})
	if err != nil || !strings.Contains(logs, "bg-process-marker") {
		t.Fatalf("logs err=%v out=%q", err, logs)
	}
	if _, err := (BgProcess{}).Run(ctx, map[string]any{"action": "kill", "job_id": id}); err != nil {
		t.Fatal(err)
	}
}

func TestBatchEditRunEntrypoint(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	path := filepath.Join(root, "batch.txt")
	if err := os.WriteFile(path, []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := (BatchEdit{}).Run(ctx, map[string]any{
		"files": []any{
			map[string]any{
				"path": path,
				"edits": []any{
					map[string]any{"old_text": "alpha", "new_text": "beta"},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(body), "beta") {
		t.Fatalf("body=%q err=%v", body, err)
	}
}

func TestUpdateActorStateRun(t *testing.T) {
	ctx := activeSessionCtx(t)
	if (UpdateActorState{}).Description() == "" || (UpdateActorState{}).Parameters() == nil {
		t.Fatal("metadata")
	}
	out, err := (UpdateActorState{}).Run(ctx, map[string]any{"key": "mood", "value": "calm"})
	if err != nil || !strings.Contains(out, "state[mood]=calm") {
		t.Fatalf("err=%v out=%q", err, out)
	}
}

func mustSession(t *testing.T) *session.Session {
	t.Helper()
	s, err := session.New(t.TempDir(), "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
