package agent

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/tools"
)

type fakeTool struct {
	name string
}

func (t fakeTool) Name() string               { return t.name }
func (t fakeTool) Description() string        { return "fake" }
func (t fakeTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (t fakeTool) Run(context.Context, map[string]any) (string, error) {
	return "ok", nil
}

type fakeMCPTool struct {
	fakeTool
	server string
}

func (t fakeMCPTool) MCPServerName() string { return t.server }

func (t fakeMCPTool) MCPLocalToolName() string { return t.fakeTool.name }

func (t fakeMCPTool) ToolNamespace() string { return t.server }

func (t fakeMCPTool) Name() string { return tools.Namespaced(t.server, t.fakeTool.name) }

func TestMCPServerOf(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(fakeTool{name: "list_dir"})
	reg.Register(fakeMCPTool{fakeTool: fakeTool{name: "query_docs"}, server: "context7"})

	if server, ok := mcpServerOf(reg, "list_dir"); ok || server != "" {
		t.Fatalf("builtin tool: expected no MCP, got server=%q ok=%v", server, ok)
	}
	if server, ok := mcpServerOf(reg, "context7__query_docs"); !ok || server != "context7" {
		t.Fatalf("mcp tool: expected context7, got server=%q ok=%v", server, ok)
	}
	if server, ok := mcpServerOf(reg, "missing"); ok || server != "" {
		t.Fatalf("unknown tool: expected no MCP, got server=%q ok=%v", server, ok)
	}
	if server, ok := mcpServerOf(nil, "list_dir"); ok || server != "" {
		t.Fatalf("nil registry: expected no MCP, got server=%q ok=%v", server, ok)
	}
}
