package config

import "strings"

func applyGlobalToolLabels(cfg *Config, global ToolLabelsUIConfig) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.UI.ToolLabels.Locale) == "" && strings.TrimSpace(global.Locale) != "" {
		cfg.UI.ToolLabels.Locale = strings.TrimSpace(global.Locale)
	}
	if len(cfg.UI.ToolLabels.Tools) == 0 && len(global.Tools) > 0 {
		cfg.UI.ToolLabels.Tools = copyStringMap(global.Tools)
	}
	if len(cfg.UI.ToolLabels.Scopes) == 0 && len(global.Scopes) > 0 {
		cfg.UI.ToolLabels.Scopes = copyStringMap(global.Scopes)
	}
	for k, v := range global.Tools {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if _, ok := cfg.UI.ToolLabels.Tools[key]; ok {
			continue
		}
		if cfg.UI.ToolLabels.Tools == nil {
			cfg.UI.ToolLabels.Tools = map[string]string{}
		}
		cfg.UI.ToolLabels.Tools[key] = strings.TrimSpace(v)
	}
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
