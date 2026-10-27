package wujipath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRootHonorsWujiRootEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WUJI_ROOT", dir)
	if got := ResolveRoot("/ignored", "../other"); got != dir {
		t.Fatalf("got %q want %q", got, dir)
	}
}

func TestModuleKindHelpers(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "wuji-core")
	taiji := filepath.Join(root, "taiji")
	wujiAI := filepath.Join(root, "wuji-ai")
	for _, pair := range []struct {
		dir, mod string
	}{
		{core, coreModule + "\n"},
		{taiji, "module github.com/coditary/taiji\n"},
		{wujiAI, "module github.com/coditary/wuji-ai\n"},
	} {
		if err := os.MkdirAll(pair.dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pair.dir, "go.mod"), []byte(pair.mod), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !isCoreModuleDir(core) || isCoreModuleDir(taiji) {
		t.Fatal("core module detection failed")
	}
	if !isTaijiModule(taiji) || !isWujiAIModule(wujiAI) {
		t.Fatal("project module detection failed")
	}
}

func TestResolveRelativeAbsolutePath(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "wuji")
	if got := resolveRelative(abs, "/cfg"); got != filepath.Clean(abs) {
		t.Fatalf("got %q", got)
	}
}
