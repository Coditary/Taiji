package config

import (
	"os"
	"path/filepath"
	"strings"
)

const memorySubdir = "memory"

// MemoryConfig controls Taiji's persistent agent memory layer.
type MemoryConfig struct {
	Enabled            *bool            `yaml:"enabled,omitempty"`
	RAGCollection      string           `yaml:"rag_collection,omitempty"`
	MinRecallScore     float32          `yaml:"min_recall_score,omitempty"`
	AutoRecallOnError  *bool            `yaml:"auto_recall_on_error,omitempty"`
	ReflectOnError     *bool            `yaml:"reflect_on_error,omitempty"`
	AutoRememberKinds  []string         `yaml:"auto_remember_kinds,omitempty"`
	GlobalDir          string           `yaml:"global_dir,omitempty"`
	SynonymsFile       string           `yaml:"synonyms_file,omitempty"`
	KindsFile          string           `yaml:"kinds_file,omitempty"`
	Kinds              []MemoryKindSpec `yaml:"kinds,omitempty"`
}

const memorySynonymsFile = "memory-synonyms.yaml"

// DefaultMemoryConfig returns built-in memory defaults.
func DefaultMemoryConfig() MemoryConfig {
	enabled := true
	autoRecall := true
	autoReflect := true
	return MemoryConfig{
		Enabled:           &enabled,
		RAGCollection:     "memory",
		MinRecallScore:    0.72,
		AutoRecallOnError: &autoRecall,
		ReflectOnError:    &autoReflect,
		AutoRememberKinds: []string{"error", "anti_pattern"},
	}
}

// ResolvedMemory merges file config with defaults.
func (c *Config) ResolvedMemory() MemoryConfig {
	out := DefaultMemoryConfig()
	if c == nil {
		return out
	}
	mergeMemory(&out, c.Memory)
	return out
}

func mergeMemory(dst *MemoryConfig, src MemoryConfig) {
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if strings.TrimSpace(src.RAGCollection) != "" {
		dst.RAGCollection = strings.TrimSpace(src.RAGCollection)
	}
	if src.MinRecallScore > 0 {
		dst.MinRecallScore = src.MinRecallScore
	}
	if src.AutoRecallOnError != nil {
		dst.AutoRecallOnError = src.AutoRecallOnError
	}
	if src.ReflectOnError != nil {
		dst.ReflectOnError = src.ReflectOnError
	}
	if len(src.AutoRememberKinds) > 0 {
		dst.AutoRememberKinds = append([]string(nil), src.AutoRememberKinds...)
	}
	if strings.TrimSpace(src.GlobalDir) != "" {
		dst.GlobalDir = strings.TrimSpace(src.GlobalDir)
	}
	if strings.TrimSpace(src.SynonymsFile) != "" {
		dst.SynonymsFile = strings.TrimSpace(src.SynonymsFile)
	}
	if strings.TrimSpace(src.KindsFile) != "" {
		dst.KindsFile = strings.TrimSpace(src.KindsFile)
	}
	if len(src.Kinds) > 0 {
		dst.Kinds = append([]MemoryKindSpec(nil), src.Kinds...)
	}
}

func (m MemoryConfig) EnabledOrDefault() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}

func (m MemoryConfig) AutoRecallOnErrorOrDefault() bool {
	if m.AutoRecallOnError == nil {
		return true
	}
	return *m.AutoRecallOnError
}

func (m MemoryConfig) ReflectOnErrorOrDefault() bool {
	if m.ReflectOnError == nil {
		return true
	}
	return *m.ReflectOnError
}

	// GlobalMemoryDir returns the user-level memory directory (~/.taiji/memory).
func GlobalMemoryDir() string {
	globalDir := GlobalConfigDir()
	if globalDir == "" {
		return ""
	}
	return filepath.Join(globalDir, memorySubdir)
}

// MemoryGlobalDir returns the configured global memory directory.
func MemoryGlobalDir(cfg *Config) string {
	if cfg != nil {
		mem := cfg.ResolvedMemory()
		if dir := strings.TrimSpace(mem.GlobalDir); dir != "" {
			return expandHome(dir)
		}
	}
	return GlobalMemoryDir()
}

// MemorySynonymsPath returns the synonym groups file for memory recall expansion.
func MemorySynonymsPath(cfg *Config) string {
	if cfg != nil {
		mem := cfg.ResolvedMemory()
		if path := strings.TrimSpace(mem.SynonymsFile); path != "" {
			return expandHome(path)
		}
	}
	globalDir := GlobalConfigDir()
	if globalDir == "" {
		return ""
	}
	return filepath.Join(globalDir, memorySynonymsFile)
}

// ProjectMemoryDir returns .taiji/memory under the project root.
func ProjectMemoryDir(root string) string {
	root = filepath.Clean(root)
	if root == "" || root == "." {
		return filepath.Join(dirName, memorySubdir)
	}
	return filepath.Join(root, dirName, memorySubdir)
}

func ensureMemoryDirs(cfg *Config) {
	if cfg == nil {
		return
	}
	_ = os.MkdirAll(MemoryGlobalDir(cfg), 0o755)
	if cfg.Root != "" {
		_ = os.MkdirAll(ProjectMemoryDir(cfg.Root), 0o755)
	}
}

func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}
