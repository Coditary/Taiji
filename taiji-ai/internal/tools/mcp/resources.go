package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coditary/taiji/internal/tools"
)

type mcpResourceDef struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Server      string
}

// ListResources returns resources from all connected MCP servers.
func (b *Bridge) ListResources(ctx context.Context) ([]mcpResourceDef, error) {
	b.mu.Lock()
	servers := make(map[string]*session, len(b.servers))
	for name, s := range b.servers {
		servers[name] = s
	}
	b.mu.Unlock()

	var out []mcpResourceDef
	for name, s := range servers {
		list, err := s.listResources(ctx)
		if err != nil {
			return nil, fmt.Errorf("mcp %q list resources: %w", name, err)
		}
		for _, r := range list {
			r.Server = name
			out = append(out, r)
		}
	}
	return out, nil
}

// ReadResource reads a resource URI from a specific MCP server.
func (b *Bridge) ReadResource(ctx context.Context, serverName, uri string) (string, error) {
	b.mu.Lock()
	s, ok := b.servers[serverName]
	b.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("mcp server %q not connected", serverName)
	}
	return s.readResource(ctx, uri)
}

// SystemPromptAppendix lists available MCP resources for the agent system prompt.
func SystemPromptAppendix(bridge *Bridge) string {
	if bridge == nil {
		return ""
	}
	list, err := bridge.ListResources(context.Background())
	if err != nil || len(list) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## MCP Resources\n")
	b.WriteString("Use read_mcp_resource with server and uri to fetch these resources:\n")
	for _, r := range list {
		line := fmt.Sprintf("- [%s] %s", r.Server, r.URI)
		if name := strings.TrimSpace(r.Name); name != "" {
			line += " (" + name + ")"
		}
		if desc := strings.TrimSpace(r.Description); desc != "" {
			line += ": " + desc
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String())
}

type listMCPResources struct {
	bridge *Bridge
}

func (t listMCPResources) Name() string { return "list_mcp_resources" }

// MCPServerName marks this as an MCP-related tool; it spans all servers.
func (t listMCPResources) MCPServerName() string { return "" }

func (t listMCPResources) MCPLocalToolName() string { return "list_mcp_resources" }

func (t listMCPResources) ToolNamespace() string { return tools.NamespaceMCPMeta }

func (t listMCPResources) Description() string {
	return "List resources exposed by connected MCP servers."
}

func (t listMCPResources) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t listMCPResources) Run(ctx context.Context, _ map[string]any) (string, error) {
	list, err := t.bridge.ListResources(ctx)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "no MCP resources available", nil
	}
	var lines []string
	for _, r := range list {
		line := fmt.Sprintf("%s %s", r.Server, r.URI)
		if name := strings.TrimSpace(r.Name); name != "" {
			line += " (" + name + ")"
		}
		if desc := strings.TrimSpace(r.Description); desc != "" {
			line += " — " + desc
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}

type readMCPResource struct {
	bridge *Bridge
}

func (t readMCPResource) Name() string { return "read_mcp_resource" }

// MCPServerName marks this as an MCP-related tool; the server is an argument.
func (t readMCPResource) MCPServerName() string { return "" }

func (t readMCPResource) MCPLocalToolName() string { return "read_mcp_resource" }

func (t readMCPResource) ToolNamespace() string { return tools.NamespaceMCPMeta }

func (t readMCPResource) Description() string {
	return "Read a resource from a connected MCP server."
}

func (t readMCPResource) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"server": map[string]any{
				"type":        "string",
				"description": "MCP server name from config",
			},
			"uri": map[string]any{
				"type":        "string",
				"description": "Resource URI from list_mcp_resources",
			},
			"download_path": map[string]any{
				"type":        "string",
				"description": "Optional workspace-relative path to save the resource instead of returning content to the model",
			},
		},
		"required": []string{"server", "uri"},
	}
}

func (t readMCPResource) Run(ctx context.Context, args map[string]any) (string, error) {
	server, _ := args["server"].(string)
	uri, _ := args["uri"].(string)
	downloadPath, _ := args["download_path"].(string)
	server = strings.TrimSpace(server)
	uri = strings.TrimSpace(uri)
	downloadPath = strings.TrimSpace(downloadPath)
	if server == "" || uri == "" {
		return "", fmt.Errorf("server and uri are required")
	}
	content, err := t.bridge.ReadResource(ctx, server, uri)
	if err != nil {
		return "", err
	}
	if downloadPath == "" {
		return content, nil
	}
	if err := os.WriteFile(downloadPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write download_path: %w", err)
	}
	return fmt.Sprintf("saved resource to %s (%d bytes)", downloadPath, len(content)), nil
}

func registerResourceTools(bridge *Bridge, reg *tools.Registry) {
	if bridge == nil || reg == nil {
		return
	}
	reg.Register(listMCPResources{bridge: bridge})
	reg.Register(readMCPResource{bridge: bridge})
}
