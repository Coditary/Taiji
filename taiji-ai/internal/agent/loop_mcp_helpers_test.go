package agent

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
	"github.com/coditary/taiji/internal/tools/mcp"
)

type stubMCPTool struct {
	builtin.ListDir
	server string
	local  string
}

func (s stubMCPTool) MCPServerName() string     { return s.server }
func (s stubMCPTool) MCPLocalToolName() string  { return s.local }
func (s stubMCPTool) Run(ctx context.Context, args map[string]any) (string, error) {
	return s.ListDir.Run(ctx, args)
}

func TestMCPServerOfAndLocalName(t *testing.T) {
	reg := tools.NewRegistry()
	if srv, ok := mcpServerOf(reg, "missing"); ok || srv != "" {
		t.Fatal("missing tool")
	}
	stub := stubMCPTool{ListDir: builtin.ListDir{}, server: "fake", local: "list_dir"}
	reg.Register(stub)
	if srv, ok := mcpServerOf(reg, "list_dir"); !ok || srv != "fake" {
		t.Fatalf("mcp server=%q ok=%v", srv, ok)
	}
	if name := mcpLocalToolName(reg, "list_dir"); name != "list_dir" {
		t.Fatalf("local name=%q", name)
	}
	_ = mcp.ServerTool(stubMCPTool{})
}

func TestToProviderToolsCopiesDefs(t *testing.T) {
	defs := []tools.Definition{{Name: "a", Description: "d", Parameters: map[string]any{"type": "object"}}}
	out := toProviderTools(defs)
	if len(out) != 1 || out[0].Name != "a" || out[0].Description != "d" {
		t.Fatalf("out=%+v", out)
	}
}
