package main

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/tools"
)

type stubTool struct {
	name string
}

func (s stubTool) Name() string { return s.name }
func (s stubTool) Description() string { return "stub" }
func (s stubTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (s stubTool) Run(context.Context, map[string]any) (string, error) { return "", nil }

func TestListBuiltinToolNamesExcludesNamespacedMCPTools(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(stubTool{name: "read_file"})
	reg.Register(stubTool{name: "context7__query-docs"})

	mcp := []mcpToolPayload{{
		Server:         "context7",
		Name:           "query-docs",
		RegisteredName: "context7__query-docs",
	}}

	got := listBuiltinToolNames(reg, mcp)
	if len(got) != 1 || got[0] != "read_file" {
		t.Fatalf("builtin tools = %v, want [read_file] only (MCP must not appear as built-in)", got)
	}
}

func TestListBuiltinToolNamesExcludesStaleMCPToolsWhenCatalogEmpty(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(stubTool{name: "grep"})
	reg.Register(stubTool{name: "context7__resolve-library-id"})

	got := listBuiltinToolNames(reg, nil)
	if len(got) != 1 || got[0] != "grep" {
		t.Fatalf("builtin tools = %v, want [grep] when MCP catalog is empty", got)
	}
}
