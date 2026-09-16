package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeWujiEndpointRelativeToConfigRoot(t *testing.T) {
	cfgRoot := filepath.Join("/tmp", "taiji-ai")
	got := normalizeWujiEndpoint("unix://../wuji-ai/.wuji/run/drivers/core.sock", cfgRoot)
	want := "unix://" + filepath.Clean(filepath.Join(cfgRoot, "../wuji-ai/.wuji/run/drivers/core.sock"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveWujiRootRelative(t *testing.T) {
	cfgRoot := filepath.Join("/tmp", "taiji-ai")
	got := resolveWujiRoot("../wuji-ai", cfgRoot)
	want := filepath.Clean(filepath.Join(cfgRoot, "../wuji-ai"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDefaultWujiCoreEndpointUsesResolvedRoot(t *testing.T) {
	root := t.TempDir()
	wujiRoot := filepath.Join(root, "wuji-ai")
	if err := os.MkdirAll(filepath.Join(wujiRoot, ".wuji", "run", "drivers"), 0o755); err != nil {
		t.Fatal(err)
	}
	taijiRoot := filepath.Join(root, "taiji-ai")
	got := defaultWujiCoreEndpoint(resolveWujiRoot("../wuji-ai", taijiRoot))
	want := unixEndpoint(filepath.Join(wujiRoot, ".wuji", "run", "drivers", "core.sock"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
