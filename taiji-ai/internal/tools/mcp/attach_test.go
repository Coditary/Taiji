package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/tools"
)

func TestAttachFromConfigNilAndEmpty(t *testing.T) {
	reg := tools.NewRegistry()
	bridge, err := AttachFromConfig(context.Background(), nil, reg)
	if err != nil || bridge == nil {
		t.Fatalf("nil cfg: bridge=%v err=%v", bridge, err)
	}
	cfg := &taijicfg.Config{}
	bridge, err = AttachFromConfig(context.Background(), cfg, reg)
	if err != nil || len(bridge.ConnectedServers()) != 0 {
		t.Fatalf("empty cfg: servers=%v err=%v", bridge.ConnectedServers(), err)
	}
}

func TestAttachFromConfigConnectsPythonServer(t *testing.T) {
	dir := t.TempDir()
	mcpFile := filepath.Join(dir, "mcp.json")
	payload, err := json.Marshal(map[string]any{
		"mcpServers": map[string]any{
			"demo": map[string]any{
				"command": "python3",
				"args":    []any{"-c", fakeMCPServer},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mcpFile, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &taijicfg.Config{
		MCP: taijicfg.MCPConfig{
			Servers:    []string{"demo"},
			ConfigPath: mcpFile,
		},
	}
	reg := tools.NewRegistry()
	bridge, err := AttachFromConfig(context.Background(), cfg, reg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bridge.Close() })
	if len(bridge.ConnectedServers()) != 1 {
		t.Fatalf("servers=%v", bridge.ConnectedServers())
	}
	if _, ok := reg.Get("demo__ping"); !ok {
		t.Fatal("expected demo__ping registered")
	}
	if _, ok := reg.Get("list_mcp_resources"); !ok {
		t.Fatal("expected resource meta tools")
	}
}

func TestLoadMCPServersInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadMCPServers(path)
	if err == nil {
		t.Fatal("expected json error")
	}
}
