package agent

import (
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func filterProviderTools(reg *tools.Registry, rt session.AgentRuntime) []provider.ToolDef {
	if reg == nil {
		return nil
	}
	defs := reg.Definitions()
	out := make([]tools.Definition, 0, len(defs))
	for _, d := range defs {
		if server, ok := mcpServerOf(reg, d.Name); ok {
			if !rt.MCPToolAllowed(server, d.Name) {
				continue
			}
		}
		out = append(out, d)
	}
	return toProviderTools(out)
}
