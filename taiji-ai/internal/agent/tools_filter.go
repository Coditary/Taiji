package agent

import (
	"strings"

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
			local := mcpLocalToolName(reg, d.Name)
			if !rt.MCPToolAllowed(server, local) {
				continue
			}
		} else if !rt.BuiltinToolAllowed(d.Name) {
			continue
		}
		out = append(out, d)
	}
	return toProviderTools(out)
}

func sessionInteractionMode(s *session.Session, rt session.AgentRuntime) string {
	if s != nil {
		if m := strings.TrimSpace(s.State.Settings.InteractionMode); m != "" {
			return m
		}
	}
	return strings.TrimSpace(rt.InteractionMode)
}

func applyInteractionModeTools(defs []provider.ToolDef, mode string) []provider.ToolDef {
	if !strings.EqualFold(strings.TrimSpace(mode), "plan") {
		return defs
	}
	var out []provider.ToolDef
	for _, d := range defs {
		if d.Name == "switch_agent_mode" {
			out = append(out, d)
		}
	}
	return out
}
