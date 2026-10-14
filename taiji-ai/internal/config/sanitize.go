package config

import (
	"os"
	"path/filepath"
	"strings"
)

var legacyWujiPathMarkers = []string{
	"Coditary-Bundle",
	"/wuji-ai/wuji-ai",
}

// sanitizeLegacyWujiPaths resets stale Wuji endpoints from old bundle checkouts.
func sanitizeLegacyWujiPaths(cfg *Config) {
	if cfg == nil {
		return
	}
	if staleWujiEndpoint(cfg.Wuji.Endpoint, cfg.Root, cfg.Wuji.WujiRoot) {
		cfg.Wuji.Endpoint = "auto"
		cfg.Wuji.WujiRoot = ""
	}
	for id, pc := range cfg.Providers {
		if pc.Type != "wuji" {
			continue
		}
		if staleWujiEndpoint(pc.Endpoint, cfg.Root, pc.WujiRoot) {
			pc.Endpoint = "auto"
			pc.WujiRoot = ""
			cfg.Providers[id] = pc
		}
	}
}

func staleWujiEndpoint(endpoint, cfgRoot, wujiRoot string) bool {
	endpoint = strings.TrimSpace(endpoint)
	wujiRoot = strings.TrimSpace(wujiRoot)
	if endpoint == "" && wujiRoot == "" {
		return false
	}
	if endpoint == "auto" && wujiRoot == "" {
		return false
	}
	if containsLegacyWujiMarker(endpoint) || containsLegacyWujiMarker(wujiRoot) {
		return true
	}
	if wujiRoot != "" && !wujiRootExists(cfgRoot, wujiRoot) {
		return true
	}
	if sock := unixSocketPath(endpoint); sock != "" && !wujiSocketRootExists(sock) {
		return true
	}
	return false
}

func containsLegacyWujiMarker(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, marker := range legacyWujiPathMarkers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func wujiRootExists(cfgRoot, wujiRoot string) bool {
	path := resolveWujiRootPath(cfgRoot, wujiRoot)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func wujiSocketRootExists(sock string) bool {
	root := wujiRootFromSocket(sock)
	if root == "" {
		return false
	}
	info, err := os.Stat(root)
	return err == nil && info.IsDir()
}

func wujiRootFromSocket(sock string) string {
	sock = filepath.Clean(sock)
	dir := filepath.Dir(sock) // drivers
	dir = filepath.Dir(dir)   // run
	dir = filepath.Dir(dir)   // .wuji
	return filepath.Dir(dir)  // project root
}

func resolveWujiRootPath(cfgRoot, wujiRoot string) string {
	wujiRoot = strings.TrimSpace(wujiRoot)
	if wujiRoot == "" {
		return ""
	}
	if filepath.IsAbs(wujiRoot) {
		return filepath.Clean(wujiRoot)
	}
	base := strings.TrimSpace(cfgRoot)
	if base == "" {
		base = "."
	}
	return filepath.Clean(filepath.Join(base, wujiRoot))
}

func unixSocketPath(endpoint string) string {
	const prefix = "unix://"
	endpoint = strings.TrimSpace(endpoint)
	if !strings.HasPrefix(endpoint, prefix) {
		return ""
	}
	return strings.TrimPrefix(endpoint, prefix)
}
