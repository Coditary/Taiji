package config

import "strings"

// SkillsRAGConfig controls semantic skill discovery at turn start.
type SkillsRAGConfig struct {
	Enabled    *bool   `yaml:"enabled,omitempty"`
	Prefetch   *bool   `yaml:"prefetch,omitempty"`
	Collection string  `yaml:"collection,omitempty"`
	TopK       int     `yaml:"top_k,omitempty"`
	MinScore   float32 `yaml:"min_score,omitempty"`
}

// DefaultSkillsRAGConfig returns built-in skill search defaults.
func DefaultSkillsRAGConfig() SkillsRAGConfig {
	enabled := true
	prefetch := true
	return SkillsRAGConfig{
		Enabled:    &enabled,
		Prefetch:   &prefetch,
		Collection: "skills",
		TopK:       5,
		MinScore:   0.55,
	}
}

// ResolvedSkillsRAG merges file config with defaults.
func (c *Config) ResolvedSkillsRAG() SkillsRAGConfig {
	out := DefaultSkillsRAGConfig()
	if c == nil {
		return out
	}
	mergeSkillsRAG(&out, c.Skills.RAG)
	return out
}

// EnabledOrDefault reports whether skill RAG search is active.
func (s SkillsRAGConfig) EnabledOrDefault() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// PrefetchOrDefault reports whether turn-start skill search should run.
func (s SkillsRAGConfig) PrefetchOrDefault() bool {
	if s.Prefetch == nil {
		return true
	}
	return *s.Prefetch
}

func mergeSkillsRAG(dst *SkillsRAGConfig, src SkillsRAGConfig) {
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if src.Prefetch != nil {
		dst.Prefetch = src.Prefetch
	}
	if strings.TrimSpace(src.Collection) != "" {
		dst.Collection = strings.TrimSpace(src.Collection)
	}
	if src.TopK > 0 {
		dst.TopK = src.TopK
	}
	if src.MinScore > 0 {
		dst.MinScore = src.MinScore
	}
}
