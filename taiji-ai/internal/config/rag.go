package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultRAGBackend           = "wuji"
	defaultRAGDriver            = "raggo"
	defaultRAGDefaultCollection = "default"
)

// RAGConfig configures Taiji's pluggable RAG layer.
type RAGConfig struct {
	Enabled           *bool  `yaml:"enabled,omitempty"`
	Backend           string `yaml:"backend,omitempty"`
	Driver            string `yaml:"driver,omitempty"`
	DefaultCollection string `yaml:"default_collection,omitempty"`
	GlobalRoot        string `yaml:"global_root,omitempty"`
}

// DefaultRAGConfig returns built-in RAG defaults.
func DefaultRAGConfig() RAGConfig {
	enabled := true
	return RAGConfig{
		Enabled:           &enabled,
		Backend:           defaultRAGBackend,
		Driver:            defaultRAGDriver,
		DefaultCollection: defaultRAGDefaultCollection,
	}
}

// ResolvedRAG merges file config with defaults.
func (c *Config) ResolvedRAG() RAGConfig {
	out := DefaultRAGConfig()
	if c == nil {
		return out
	}
	mergeRAG(&out, c.RAG)
	return out
}

func mergeRAG(dst *RAGConfig, src RAGConfig) {
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if strings.TrimSpace(src.Backend) != "" {
		dst.Backend = strings.TrimSpace(src.Backend)
	}
	if strings.TrimSpace(src.Driver) != "" {
		dst.Driver = strings.TrimSpace(src.Driver)
	}
	if strings.TrimSpace(src.DefaultCollection) != "" {
		dst.DefaultCollection = strings.TrimSpace(src.DefaultCollection)
	}
	if strings.TrimSpace(src.GlobalRoot) != "" {
		dst.GlobalRoot = strings.TrimSpace(src.GlobalRoot)
	}
}

func (r RAGConfig) EnabledOrDefault() bool {
	if r.Enabled == nil {
		return true
	}
	return *r.Enabled
}

// GlobalStoreRoot returns the Wuji project root used for shared RAG collections.
func (r RAGConfig) GlobalStoreRoot(localRoot, wujiRoot string) string {
	if strings.TrimSpace(r.GlobalRoot) != "" {
		root := strings.TrimSpace(r.GlobalRoot)
		if filepath.IsAbs(root) {
			return filepath.Clean(root)
		}
		base := strings.TrimSpace(localRoot)
		if base == "" {
			base = "."
		}
		return filepath.Clean(filepath.Join(base, root))
	}
	return resolveWujiProjectRoot(localRoot, wujiRoot)
}

func resolveWujiProjectRoot(localRoot, wujiRoot string) string {
	if v := strings.TrimSpace(os.Getenv("WUJI_ROOT")); v != "" {
		return filepath.Clean(v)
	}
	root := strings.TrimSpace(wujiRoot)
	base := strings.TrimSpace(localRoot)
	if base == "" {
		base = "."
	}
	if root == "" {
		root = "../wuji-ai/wuji-core"
	}
	if filepath.IsAbs(root) {
		return filepath.Clean(root)
	}
	return filepath.Clean(filepath.Join(base, root))
}
