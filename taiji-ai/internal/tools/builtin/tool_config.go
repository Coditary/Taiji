package builtin

import (
	taijicfg "github.com/coditary/taiji/internal/config"
)

var toolsSettings = taijicfg.DefaultToolsConfig()

// ConfigureTools applies Taiji config to built-in tool behavior.
func ConfigureTools(cfg *taijicfg.Config) {
	if cfg == nil {
		toolsSettings = taijicfg.DefaultToolsConfig()
		return
	}
	toolsSettings = cfg.ResolvedTools()
}
