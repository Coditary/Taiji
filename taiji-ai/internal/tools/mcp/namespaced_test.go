package mcp

import (
	"testing"

	"github.com/coditary/taiji/internal/tools"
)

func TestMCPToolRegisteredName(t *testing.T) {
	tool := &mcpTool{
		server: "exa",
		def:    mcpToolDef{Name: "web_search"},
	}
	if tool.Name() != "exa__web_search" {
		t.Fatalf("name: got %q", tool.Name())
	}
	if tool.MCPLocalToolName() != "web_search" {
		t.Fatalf("local: got %q", tool.MCPLocalToolName())
	}
	if tool.ToolNamespace() != "exa" {
		t.Fatalf("namespace: got %q", tool.ToolNamespace())
	}
	if ns, local, ok := tools.SplitNamespaced(tool.Name()); !ok || ns != "exa" || local != "web_search" {
		t.Fatalf("split: ns=%q local=%q ok=%v", ns, local, ok)
	}
	if tool.MCPServerName() != "exa" || tool.Description() != "" {
		t.Fatalf("server/desc: %q %q", tool.MCPServerName(), tool.Description())
	}
	schema := tool.Parameters()
	if schema["type"] != "object" {
		t.Fatalf("params=%v", schema)
	}
	tool.def.Description = "Search the web"
	tool.def.InputSchema = map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}}}
	if tool.Description() != "Search the web" || tool.Parameters()["properties"] == nil {
		t.Fatal("expected def fields on tool")
	}
}
