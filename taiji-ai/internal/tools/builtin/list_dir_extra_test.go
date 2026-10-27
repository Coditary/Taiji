package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestListDirRecursiveAndNotDirectory(t *testing.T) {
	root, ctx := useProjectWorkspace(t)
	sub := filepath.Join(root, "nested", "child")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "leaf.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (ListDir{}).Run(ctx, map[string]any{
		"path":      sub,
		"recursive": true,
		"max_depth": 2,
	})
	if err != nil || !strings.Contains(out, "leaf.txt") {
		t.Fatalf("recursive: err=%v out=%q", err, out)
	}
	file := filepath.Join(root, "only.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = (ListDir{}).Run(ctx, map[string]any{"path": file})
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("err=%v", err)
	}
}

func TestConfigureToolsNilAndCustom(t *testing.T) {
	ConfigureTools(nil)
	ConfigureTools(&taijicfg.Config{Root: t.TempDir()})
}
