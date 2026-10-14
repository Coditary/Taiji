package wujipath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRootConfiguredRelative(t *testing.T) {
	cfgRoot := filepath.Join("/tmp", "taiji")
	got := ResolveRoot(cfgRoot, "../wuji")
	want := filepath.Clean(filepath.Join(cfgRoot, "../wuji"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveCoreModuleDirFromTree(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "wuji-core")
	if err := os.MkdirAll(core, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(core, "go.mod"), []byte(coreModule+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WUJI_ROOT", core)
	dir, err := ResolveCoreModuleDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != core {
		t.Fatalf("got %q want %q", dir, core)
	}
}
