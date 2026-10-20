package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBatchEditAtomicValidation(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	b := filepath.Join(root, "b.txt")
	_ = os.WriteFile(a, []byte("alpha\n"), 0o644)
	_ = os.WriteFile(b, []byte("beta\n"), 0o644)

	_, err := (BatchEdit{}).RunWithMetadata(context.Background(), map[string]any{
		"files": []any{
			map[string]any{
				"path": a,
				"edits": []any{map[string]any{"old_text": "alpha", "new_text": "ALPHA"}},
			},
			map[string]any{
				"path": b,
				"edits": []any{map[string]any{"old_text": "missing", "new_text": "X"}},
			},
		},
	})
	if err == nil {
		t.Fatal("expected validation failure before writes")
	}
	raw, _ := os.ReadFile(a)
	if string(raw) != "alpha\n" {
		t.Fatalf("a should be unchanged: %q", raw)
	}
}

func TestBatchEditAppliesAll(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	b := filepath.Join(root, "b.txt")
	_ = os.WriteFile(a, []byte("alpha\n"), 0o644)
	_ = os.WriteFile(b, []byte("beta\n"), 0o644)

	res, err := (BatchEdit{}).RunWithMetadata(context.Background(), map[string]any{
		"files": []any{
			map[string]any{"path": a, "edits": []any{map[string]any{"old_text": "alpha", "new_text": "ALPHA"}}},
			map[string]any{"path": b, "edits": []any{map[string]any{"old_text": "beta", "new_text": "BETA"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "batch_edit applied to 2") {
		t.Fatalf("text=%q", res.Text)
	}
}
