package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/tools"
)

const fakeMCPWithResources = `
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
        print(json.dumps({"jsonrpc":"2.0","id":mid,"result":{"tools":[]}}))
        sys.stdout.flush()
    elif method == "resources/list":
        print(json.dumps({"jsonrpc":"2.0","id":mid,"result":{"resources":[{"uri":"file:///doc","name":"Doc","description":"A doc","mimeType":"text/plain"}]}}))
        sys.stdout.flush()
    elif method == "resources/read":
        print(json.dumps({"jsonrpc":"2.0","id":mid,"result":{"contents":[{"type":"text","text":"hello resource"}]}}))
        sys.stdout.flush()
`

func connectFakeResourceServer(t *testing.T) *Bridge {
	t.Helper()
	b := NewBridge()
	t.Cleanup(func() { _ = b.Close() })
	if err := b.Connect(context.Background(), ServerConfig{
		Name:    "resfake",
		Command: "python3",
		Args:    []string{"-c", fakeMCPWithResources},
	}); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBridgeListAndReadResources(t *testing.T) {
	b := connectFakeResourceServer(t)
	if names := b.ConnectedServers(); len(names) != 1 || names[0] != "resfake" {
		t.Fatalf("servers=%v", names)
	}
	list, err := b.ListResources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].URI != "file:///doc" || list[0].Server != "resfake" {
		t.Fatalf("list=%+v", list)
	}
	body, err := b.ReadResource(context.Background(), "resfake", "file:///doc")
	if err != nil || body != "hello resource" {
		t.Fatalf("read: err=%v body=%q", err, body)
	}
	_, err = b.ReadResource(context.Background(), "missing", "file:///doc")
	if err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("expected missing server error, got %v", err)
	}
}

func TestSystemPromptAppendixResources(t *testing.T) {
	b := connectFakeResourceServer(t)
	appendix := SystemPromptAppendix(b)
	if !strings.Contains(appendix, "MCP Resources") || !strings.Contains(appendix, "file:///doc") {
		t.Fatalf("appendix=%q", appendix)
	}
	if SystemPromptAppendix(nil) != "" {
		t.Fatal("nil bridge should be empty")
	}
}

func TestResourceToolsRun(t *testing.T) {
	b := connectFakeResourceServer(t)
	reg := tools.NewRegistry()
	registerResourceTools(b, reg)

	listTool, ok := reg.Get("list_mcp_resources")
	if !ok {
		t.Fatal("list tool missing")
	}
	out, err := listTool.Run(context.Background(), map[string]any{})
	if err != nil || !strings.Contains(out, "file:///doc") {
		t.Fatalf("list: err=%v out=%q", err, out)
	}

	readTool, ok := reg.Get("read_mcp_resource")
	if !ok {
		t.Fatal("read tool missing")
	}
	content, err := readTool.Run(context.Background(), map[string]any{
		"server": "resfake",
		"uri":    "file:///doc",
	})
	if err != nil || content != "hello resource" {
		t.Fatalf("read: err=%v out=%q", err, content)
	}

	dir := t.TempDir()
	dl := filepath.Join(dir, "saved.txt")
	saved, err := readTool.Run(context.Background(), map[string]any{
		"server":        "resfake",
		"uri":           "file:///doc",
		"download_path": dl,
	})
	if err != nil || !strings.Contains(saved, "saved resource") {
		t.Fatalf("download: err=%v out=%q", err, saved)
	}
	raw, err := os.ReadFile(dl)
	if err != nil || string(raw) != "hello resource" {
		t.Fatalf("file: err=%v raw=%q", err, raw)
	}

	_, err = readTool.Run(context.Background(), map[string]any{"server": "resfake"})
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestListMCPResourcesMetadata(t *testing.T) {
	tool := listMCPResources{}
	if tool.Name() != "list_mcp_resources" || tool.ToolNamespace() != tools.NamespaceMCPMeta {
		t.Fatalf("meta: name=%q ns=%q", tool.Name(), tool.ToolNamespace())
	}
	if tool.Parameters()["type"] != "object" {
		t.Fatal("expected object schema")
	}
}
