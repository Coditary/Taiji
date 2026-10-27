package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFindTaijiConfigRootWalksUp(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(filepath.Join(root, dirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, dirName, configFile), []byte("default_provider: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := findTaijiConfigRoot(nested)
	if !ok || got != root {
		t.Fatalf("root = %q ok=%v", got, ok)
	}
}

func TestGlobalMCPServerNamesAndSearchPaths(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	globalDir := GlobalConfigDir()
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	globalYAML := `mcp:
  servers: [global-a, global-b]
`
	if err := os.WriteFile(filepath.Join(globalDir, configFile), []byte(globalYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	names := GlobalMCPServerNames()
	if len(names) != 2 || names[0] != "global-a" {
		t.Fatalf("names = %#v", names)
	}

	project := t.TempDir()
	projectMCP := filepath.Join(project, ".wuji", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(projectMCP), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectMCP, []byte(`{"servers":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := defaultConfig(project)
	cfg.MCP.ConfigPath = projectMCP
	paths := MCPConfigSearchPaths(cfg)
	if len(paths) == 0 || paths[0] != projectMCP {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestHookSearchPathsIncludesExistingFile(t *testing.T) {
	root := t.TempDir()
	hookFile := filepath.Join(root, "hooks.yaml")
	if err := os.WriteFile(hookFile, []byte("hooks: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := defaultConfig(root)
	cfg.Hooks.Paths = []string{hookFile, filepath.Join(root, "missing.yaml")}
	paths := HookSearchPaths(cfg)
	if len(paths) != 1 || paths[0] != hookFile {
		t.Fatalf("paths = %#v", paths)
	}
	disabled := false
	cfg.Hooks.Enabled = &disabled
	if cfg.Hooks.HooksEnabled() {
		t.Fatal("expected hooks disabled")
	}
	_ = HookSearchPaths(nil)
}

func TestChainOfThoughtModeYAML(t *testing.T) {
	var mode ChainOfThoughtMode
	if err := yaml.Unmarshal([]byte("full"), &mode); err != nil || !mode.UsesFullPass() {
		t.Fatalf("full mode: %q err=%v", mode, err)
	}
	out, err := mode.MarshalYAML()
	if err != nil || out != "full" {
		t.Fatalf("marshal = %v err=%v", out, err)
	}
	var off ChainOfThoughtMode
	if err := yaml.Unmarshal([]byte("off"), &off); err != nil || off.UsesFullPass() {
		t.Fatalf("off mode: %q err=%v", off, err)
	}
}
