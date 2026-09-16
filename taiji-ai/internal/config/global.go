package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const globalMCPFile = "mcp.json"

// GlobalConfigDir returns the user-level Taiji config directory (~/.taiji).
func GlobalConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, dirName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, dirName)
}

func loadGlobalFileConfig() *fileConfig {
	globalDir := GlobalConfigDir()
	if globalDir == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(globalDir, configFile))
	if err != nil {
		return nil
	}
	var global fileConfig
	if err := yaml.Unmarshal(raw, &global); err != nil {
		return nil
	}
	return &global
}

// GlobalMCPServerNames returns MCP server names from ~/.taiji/config.yaml.
func GlobalMCPServerNames() []string {
	global := loadGlobalFileConfig()
	if global == nil {
		return nil
	}
	return global.MCP.Servers
}

// MCPConfigSearchPaths returns MCP definition files to load, project first then global.
func MCPConfigSearchPaths(cfg *Config) []string {
	seen := map[string]struct{}{}
	var paths []string

	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || !fileExists(path) {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}

	if cfg != nil {
		add(cfg.MCP.ConfigPath)
	}

	globalDir := GlobalConfigDir()
	if globalDir != "" {
		for _, candidate := range globalMCPConfigCandidates(globalDir) {
			add(candidate)
		}
	}

	return paths
}

func applyGlobalMCPDefaults(cfg *Config) {
	if cfg == nil {
		return
	}

	global := loadGlobalFileConfig()
	if global != nil {
		if strings.TrimSpace(cfg.DefaultProvider) == "" && strings.TrimSpace(global.DefaultProvider) != "" {
			cfg.DefaultProvider = global.DefaultProvider
		}
		if len(cfg.Providers) == 0 && len(global.Providers) > 0 {
			cfg.Providers = global.Providers
		}
		if isDefaultWujiConfig(cfg.Wuji) && !isDefaultWujiConfig(global.Wuji) {
			cfg.Wuji = global.Wuji
		}
		cfg.MCP.Servers = mergeUnique(global.MCP.Servers, cfg.MCP.Servers)
	} else {
		cfg.MCP.Servers = mergeUnique(GlobalMCPServerNames(), cfg.MCP.Servers)
	}

	if fileExists(cfg.MCP.ConfigPath) {
		return
	}

	for _, candidate := range globalMCPConfigCandidates(GlobalConfigDir()) {
		if fileExists(candidate) {
			cfg.MCP.ConfigPath = candidate
			return
		}
	}
}

func isDefaultWujiConfig(wuji WujiConfig) bool {
	return !wuji.AutoSync &&
		strings.TrimSpace(wuji.Endpoint) == "auto" &&
		strings.TrimSpace(wuji.WujiRoot) == "../wuji-ai/wuji-core"
}

func globalMCPConfigCandidates(globalDir string) []string {
	if globalDir == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(globalDir, globalMCPFile),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".cursor", globalMCPFile))
	}
	return candidates
}

func mergeUnique(primary, secondary []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(primary)+len(secondary))
	for _, name := range primary {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, name := range secondary {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
