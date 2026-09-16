package provider

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultWujiCoreEndpoint(wujiRoot string) string {
	if v := strings.TrimSpace(os.Getenv("WUJI_CORE_ADDR")); v != "" {
		return normalizeWujiEndpoint(v, "")
	}
	sock := filepath.Join(resolveWujiRoot(wujiRoot, "."), ".wuji", "run", "drivers", "core.sock")
	return unixEndpoint(sock)
}

func resolveWujiRoot(wujiRoot, cfgRoot string) string {
	if v := strings.TrimSpace(os.Getenv("WUJI_ROOT")); v != "" {
		return filepath.Clean(v)
	}
	root := strings.TrimSpace(wujiRoot)
	if root == "" {
		root = strings.TrimSpace(cfgRoot)
	}
	if root == "" {
		root = "."
	}
	if filepath.IsAbs(root) {
		return filepath.Clean(root)
	}
	base := strings.TrimSpace(cfgRoot)
	if base == "" {
		base = "."
	}
	return filepath.Clean(filepath.Join(base, root))
}

func normalizeWujiEndpoint(endpoint, cfgRoot string) string {
	ep := strings.TrimSpace(endpoint)
	if ep == "" {
		return ep
	}
	const prefix = "unix://"
	if !strings.HasPrefix(ep, prefix) {
		return ep
	}
	path := strings.TrimPrefix(ep, prefix)
	if filepath.IsAbs(path) {
		return prefix + filepath.Clean(path)
	}
	base := strings.TrimSpace(cfgRoot)
	if base == "" {
		base = "."
	}
	return unixEndpoint(filepath.Clean(filepath.Join(base, path)))
}

func unixEndpoint(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return "unix://" + abs
}
