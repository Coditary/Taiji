package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	cfg := defaultConfig(dir)
	cfg.DefaultProvider = "vllm"
	cfg.Providers = map[string]ProviderConfig{
		"vllm": {
			Type:       "wuji",
			ProviderID: "vllm",
			Model:      "meta-llama/Llama-3.2-1B",
			Synced:     true,
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultProvider != "vllm" {
		t.Fatalf("default: %q", loaded.DefaultProvider)
	}
	if loaded.Providers["vllm"].Model != "meta-llama/Llama-3.2-1B" {
		t.Fatalf("providers: %+v", loaded.Providers)
	}
}

func TestRootLocalUsesCWDNotParentProject(t *testing.T) {
	project := t.TempDir()
	runtimeDir := filepath.Join(project, dirName)
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, configFile), []byte("default_provider: parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(project, "sub", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	root, err := Root(LoadOptions{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	absNested, err := filepath.Abs(nested)
	if err != nil {
		t.Fatal(err)
	}
	if root != absNested {
		t.Fatalf("root = %q, want %q", root, absNested)
	}

	root, ok := findProjectRoot(nested)
	if !ok || root != project {
		t.Fatalf("non-local walk-up = %q, want %q", root, project)
	}
}

func TestRootFindsTaijiConfigWalkingUp(t *testing.T) {
	project := t.TempDir()
	runtimeDir := filepath.Join(project, dirName)
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, configFile), []byte("default_provider: gemma4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(project, "sub", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	root, ok := findProjectRoot(nested)
	if !ok {
		t.Fatal("expected project root")
	}
	if root != project {
		t.Fatalf("root = %q, want %q", root, project)
	}
}

func TestApplyGlobalMCPDefaultsAppliesProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	globalDir := filepath.Join(home, dirName)
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, configFile), []byte(`default_provider: gemma4
providers:
  gemma4:
    type: wuji
    provider_id: llama
    model: test-model
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := defaultConfig(t.TempDir())
	applyGlobalMCPDefaults(cfg)

	if cfg.DefaultProvider != "gemma4" {
		t.Fatalf("default provider: %q", cfg.DefaultProvider)
	}
	if cfg.Providers["gemma4"].Model != "test-model" {
		t.Fatalf("providers: %+v", cfg.Providers)
	}
}

func TestApplyGlobalMCPDefaultsMergesWithProject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	globalDir := filepath.Join(home, dirName)
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, configFile), []byte(`mcp:
  servers:
    - tempify
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := defaultConfig(t.TempDir())
	cfg.MCP.Servers = []string{"context7"}
	applyGlobalMCPDefaults(cfg)

	if len(cfg.MCP.Servers) != 2 || cfg.MCP.Servers[0] != "tempify" || cfg.MCP.Servers[1] != "context7" {
		t.Fatalf("servers: %+v", cfg.MCP.Servers)
	}
}

func TestApplyGlobalMCPDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	globalDir := filepath.Join(home, dirName)
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, configFile), []byte(`mcp:
  servers:
    - tempify
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "mcp.json"), []byte(`{"mcpServers":{"tempify":{"command":"mcp-tempify"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := defaultConfig(t.TempDir())
	applyGlobalMCPDefaults(cfg)

	if len(cfg.MCP.Servers) != 1 || cfg.MCP.Servers[0] != "tempify" {
		t.Fatalf("servers: %+v", cfg.MCP.Servers)
	}
	wantPath := filepath.Join(globalDir, "mcp.json")
	if cfg.MCP.ConfigPath != wantPath {
		t.Fatalf("config path = %q, want %q", cfg.MCP.ConfigPath, wantPath)
	}
}

func TestAutoNameSessionsEnabledDefault(t *testing.T) {
	if !(AgentConfig{}).AutoNameSessionsEnabled() {
		t.Fatal("expected auto naming enabled by default")
	}
	falseVal := false
	if (AgentConfig{AutoNameSessions: &falseVal}).AutoNameSessionsEnabled() {
		t.Fatal("expected auto naming disabled when configured false")
	}
}
