package config

import (
	"os"
	"path/filepath"
	"strings"
)

// BundledTemplatesDir returns the shipped markdown/yaml template directory when present.
func BundledTemplatesDir(projectRoot string) string {
	seen := map[string]struct{}{}
	var candidates []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}

	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot != "" {
		add(filepath.Join(projectRoot, "taiji-ai", "global", "templates"))
		add(filepath.Join(projectRoot, "global", "templates"))
	}
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 12; i++ {
			add(filepath.Join(dir, "taiji-ai", "global", "templates"))
			add(filepath.Join(dir, "global", "templates"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 8; i++ {
			add(filepath.Join(dir, "share", "taiji", "templates"))
			add(filepath.Join(dir, "taiji-ai", "global", "templates"))
			add(filepath.Join(dir, "global", "templates"))
			add(filepath.Join(dir, "..", "taiji-ai", "global", "templates"))
			add(filepath.Join(dir, "..", "share", "taiji", "templates"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if global := GlobalConfigDir(); global != "" {
		add(filepath.Join(global, "templates"))
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return ""
}
