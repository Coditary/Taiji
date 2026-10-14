package config

import (
	"os"
	"path/filepath"
	"strings"
)

const memoryKindsFile = "memory-kinds.yaml"

// MemoryKindSpec defines a configurable memory kind (inline in config or YAML file).
type MemoryKindSpec struct {
	Name                string   `yaml:"name"`
	Description         string   `yaml:"description,omitempty"`
	StoreWhen           string   `yaml:"store_when,omitempty"`
	RecallWhen          string   `yaml:"recall_when,omitempty"`
	RequiresVerifiedFix *bool    `yaml:"requires_verified_fix,omitempty"`
	FailureKind         *bool    `yaml:"failure_kind,omitempty"`
	AutoFingerprint     *bool    `yaml:"auto_fingerprint,omitempty"`
	EnrichName          *bool    `yaml:"enrich_name,omitempty"`
	RecallHints         []string `yaml:"recall_hints,omitempty"`
	StoreHints          []string `yaml:"store_hints,omitempty"`
}

// MemoryKindsSearchPaths returns memory-kinds.yaml files to load, lowest to highest priority.
// Later files override kinds with the same name; inline config kinds win last.
func MemoryKindsSearchPaths(cfg *Config) []string {
	seen := map[string]struct{}{}
	var paths []string

	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}

	if cfg != nil {
		if bundled := BundledMemoryKindsPath(cfg.Root); bundled != "" {
			add(bundled)
		}
	} else if bundled := BundledMemoryKindsPath(""); bundled != "" {
		add(bundled)
	}

	globalDir := GlobalConfigDir()
	if globalDir != "" {
		add(filepath.Join(globalDir, memoryKindsFile))
	}

	if cfg != nil && cfg.Root != "" {
		add(filepath.Join(cfg.Root, dirName, memoryKindsFile))
		if path := strings.TrimSpace(cfg.ResolvedMemory().KindsFile); path != "" {
			add(expandHome(path))
		}
	}

	return paths
}

// BundledMemoryKindsPath returns taiji-ai/global/memory-kinds.yaml when present.
func BundledMemoryKindsPath(projectRoot string) string {
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
		add(filepath.Join(projectRoot, "taiji-ai", "global", memoryKindsFile))
		add(filepath.Join(projectRoot, "global", memoryKindsFile))
	}
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 12; i++ {
			add(filepath.Join(dir, "taiji-ai", "global", memoryKindsFile))
			add(filepath.Join(dir, "global", memoryKindsFile))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 4; i++ {
			add(filepath.Join(dir, "global", memoryKindsFile))
			add(filepath.Join(dir, "taiji-ai", "global", memoryKindsFile))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
