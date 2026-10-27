package mcp

import (
	"context"
	"os/exec"
	"testing"

	"github.com/coditary/taiji/internal/tools"
)

const fakeMCPServer = `
import json, sys
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    msg = json.loads(line)
    mid = msg.get("id")
    method = msg.get("method")
    if method == "initialize":
        print(json.dumps({"jsonrpc":"2.0","id":mid,"result":{"protocolVersion":"2024-11-05","capabilities":{}}}))
        sys.stdout.flush()
    elif method == "notifications/initialized":
        continue
    elif method == "tools/list":
        print(json.dumps({"jsonrpc":"2.0","id":mid,"result":{"tools":[{"name":"ping","description":"Ping tool","inputSchema":{"type":"object"}}]}}))
        sys.stdout.flush()
    elif method == "tools/call":
        print(json.dumps({"jsonrpc":"2.0","id":mid,"result":{"content":[{"type":"text","text":"pong"}]}}))
        sys.stdout.flush()
`

func TestBridgeConnectListAndCallTool(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required for fake MCP server")
	}
	b := NewBridge()
	t.Cleanup(func() { _ = b.Close() })
	err := b.Connect(context.Background(), ServerConfig{
		Name:    "fake",
		Command: "python3",
		Args:    []string{"-c", fakeMCPServer},
	})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := b.ToolCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 1 || catalog[0].Name != "ping" {
		t.Fatalf("catalog=%v", catalog)
	}
	reg := tools.NewRegistry()
	if err := b.RegisterTools(reg, "fake"); err != nil {
		t.Fatal(err)
	}
	tool, ok := reg.Get("fake__ping")
	if !ok {
		t.Fatal("tool not registered")
	}
	out, err := tool.Run(context.Background(), map[string]any{})
	if err != nil || out != "pong" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}
