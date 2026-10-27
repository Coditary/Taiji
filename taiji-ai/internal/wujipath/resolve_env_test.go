package wujipath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRootUsesWujiRootEnv(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom-wuji")
	t.Setenv("WUJI_ROOT", want)
	got := ResolveRoot("/ignored", "../relative")
	if got != filepath.Clean(want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestModuleDirFromEnvWujiCoreModule(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(coreModule+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WUJI_CORE_MODULE", root)
	dir, ok := moduleDirFromEnv()
	if !ok || dir != filepath.Clean(root) {
		t.Fatalf("dir=%q ok=%v", dir, ok)
	}
}
