package runtime

import (
	"context"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func testRuntimeConfig(t *testing.T) *taijicfg.Config {
	root := t.TempDir()
	return &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy": {Type: "dummy", Model: "alpha"},
		},
		Wuji: taijicfg.WujiConfig{AutoSync: false},
		Skills: taijicfg.SkillsConfig{
			Dirs: []string{filepath.Join(root, ".taiji", "skills")},
		},
		MCP: taijicfg.MCPConfig{
			ConfigPath: filepath.Join(root, ".wuji", "mcp.json"),
		},
	}
}

func TestNewWithDummyProvider(t *testing.T) {
	cfg := testRuntimeConfig(t)
	rt, err := New(context.Background(), cfg, Options{Quiet: true, Local: true, LocalOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if rt == nil || rt.Providers == nil || rt.Tools == nil {
		t.Fatal("incomplete runtime")
	}
	p, err := rt.Providers.Default()
	if err != nil {
		t.Fatal(err)
	}
	if p.ID() != "dummy" {
		t.Fatalf("provider=%q", p.ID())
	}
	if err := rt.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewNilConfigReturnsError(t *testing.T) {
	_, err := New(context.Background(), nil, Options{Quiet: true})
	if err == nil {
		t.Fatal("expected error when config is nil")
	}
}
