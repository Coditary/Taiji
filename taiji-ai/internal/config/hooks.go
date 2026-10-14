package config

import (
	"os"
	"strings"

	"github.com/coditary/taiji/internal/hooks"
)

// HooksConfig controls lifecycle hook loading.
type HooksConfig struct {
	Enabled *bool    `yaml:"enabled,omitempty"`
	Paths   []string `yaml:"paths,omitempty"`
}

// HooksEnabled reports whether hooks are active (default true).
func (h HooksConfig) HooksEnabled() bool {
	if h.Enabled == nil {
		return true
	}
	return *h.Enabled
}

// HookSearchPaths returns existing hook files to load.
func HookSearchPaths(cfg *Config) []string {
	seen := map[string]struct{}{}
	var paths []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		if _, err := os.Stat(path); err != nil {
			return
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	if cfg != nil {
		for _, path := range cfg.Hooks.Paths {
			add(path)
		}
		root := cfg.Root
		for _, path := range hooks.DefaultSearchPaths(root) {
			add(path)
		}
	} else {
		for _, path := range hooks.DefaultSearchPaths("") {
			add(path)
		}
	}
	return paths
}
