package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestManageFileDelete(t *testing.T) {
	root := t.TempDir()
	SetWorkspaceRootsForTest(root, filepath.Join(root, "sessions"))
	path := filepath.Join(root, "gone.txt")
	if err := os.WriteFile(path, []byte("bye"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := WithToolSessionID(context.Background(), "mf")
	res, err := (ManageFile{}).RunWithMetadata(ctx, map[string]any{"action": "delete", "path": path})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}
	if res.Text == "" {
		t.Fatal("expected message")
	}
}
