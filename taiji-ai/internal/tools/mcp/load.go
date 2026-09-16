package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/tools"
)

type mcpFile struct {
	MCPServers map[string]mcpServerDef `json:"mcpServers"`
}

type mcpServerDef struct {
	Command  string            `json:"command"`
	Args     []string          `json:"args"`
	Env      map[string]string `json:"env"`
	Cwd      string            `json:"cwd"`
	URL      string            `json:"url"`
	Disabled bool              `json:"disabled"`
}

// AttachFromConfig connects configured MCP servers and registers their tools.
func AttachFromConfig(ctx context.Context, cfg *taijicfg.Config, reg *tools.Registry) (*Bridge, error) {
	bridge := NewBridge()
	if cfg == nil || len(cfg.MCP.Servers) == 0 {
		return bridge, nil
	}

	servers, err := loadMCPServersFromPaths(taijicfg.MCPConfigSearchPaths(cfg))
	if err != nil {
		return bridge, err
	}

	for _, name := range cfg.MCP.Servers {
		def, ok := servers[name]
		if !ok {
			return bridge, fmt.Errorf("mcp server %q not found in MCP config search paths", name)
		}
		if def.Disabled {
			continue
		}
		if strings.TrimSpace(def.URL) != "" {
			return bridge, fmt.Errorf("mcp server %q uses url transport (not supported in MVP)", name)
		}
		sc := ServerConfig{Name: name, Command: def.Command, Args: def.Args, Env: def.Env, Cwd: def.Cwd}
		if err := bridge.Connect(ctx, sc); err != nil {
			return bridge, fmt.Errorf("connect mcp %q: %w", name, err)
		}
		if err := bridge.RegisterTools(reg, name); err != nil {
			return bridge, fmt.Errorf("register mcp tools %q: %w", name, err)
		}
	}
	if len(cfg.MCP.Servers) > 0 {
		registerResourceTools(bridge, reg)
	}
	return bridge, nil
}

func loadMCPServersFromPaths(paths []string) (map[string]mcpServerDef, error) {
	merged := map[string]mcpServerDef{}
	for _, path := range paths {
		defs, err := loadMCPServers(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for name, def := range defs {
			merged[name] = def
		}
	}
	return merged, nil
}

func loadMCPServers(path string) (map[string]mcpServerDef, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file mcpFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, err
	}
	if file.MCPServers == nil {
		return map[string]mcpServerDef{}, nil
	}
	return file.MCPServers, nil
}
