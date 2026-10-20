package agent

import (
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestTodoPlanAppendixOpenItems(t *testing.T) {
	builtin.SetWorkspaceRootsForTest(t.TempDir(), t.TempDir())
	_, err := (builtin.TodoPlanTool{}).Run(builtin.WithToolSessionID(t.Context(), "s1"), map[string]any{
		"action": "set",
		"items": []any{
			map[string]any{"id": "1", "task": "ship feature", "status": "in_progress"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := todoPlanAppendix("s1")
	if !strings.Contains(out, "Session todo plan") || !strings.Contains(out, "ship feature") {
		t.Fatalf("appendix=%q", out)
	}
}
