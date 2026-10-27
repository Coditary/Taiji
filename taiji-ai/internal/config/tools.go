package config

import (
	"os"
	"strings"
)

// ToolsConfig controls built-in agent tool behavior.
type ToolsConfig struct {
	AutoFormat          *bool  `yaml:"auto_format,omitempty"`
	AutoVet             *bool  `yaml:"auto_vet,omitempty"`
	AllowLocalFetch     *bool  `yaml:"allow_local_fetch,omitempty"`
	BraveSearchAPIKey   string `yaml:"brave_search_api_key,omitempty"`
	WebSearchMaxResults int    `yaml:"web_search_max_results,omitempty"`
}

// DefaultToolsConfig returns built-in tool defaults.
func DefaultToolsConfig() ToolsConfig {
	autoFormat := true
	autoVet := true
	return ToolsConfig{
		AutoFormat: &autoFormat,
		AutoVet:    &autoVet,
	}
}

// ResolvedTools merges file config with defaults.
func (c *Config) ResolvedTools() ToolsConfig {
	out := DefaultToolsConfig()
	if c == nil {
		return out
	}
	mergeTools(&out, c.Tools)
	return out
}

func mergeTools(dst *ToolsConfig, src ToolsConfig) {
	if src.AutoFormat != nil {
		dst.AutoFormat = src.AutoFormat
	}
	if src.AutoVet != nil {
		dst.AutoVet = src.AutoVet
	}
	if src.AllowLocalFetch != nil {
		dst.AllowLocalFetch = src.AllowLocalFetch
	}
	if strings.TrimSpace(src.BraveSearchAPIKey) != "" {
		dst.BraveSearchAPIKey = strings.TrimSpace(src.BraveSearchAPIKey)
	}
	if src.WebSearchMaxResults > 0 {
		dst.WebSearchMaxResults = src.WebSearchMaxResults
	}
}

func (t ToolsConfig) AutoFormatOrDefault() bool {
	if t.AutoFormat == nil {
		return true
	}
	return *t.AutoFormat
}

func (t ToolsConfig) AutoVetOrDefault() bool {
	if t.AutoVet == nil {
		return true
	}
	return *t.AutoVet
}

func (t ToolsConfig) AllowLocalFetchOrDefault() bool {
	if t.AllowLocalFetch == nil {
		return false
	}
	return *t.AllowLocalFetch
}

func (t ToolsConfig) WebSearchMaxResultsOrDefault() int {
	if t.WebSearchMaxResults <= 0 {
		return 8
	}
	return t.WebSearchMaxResults
}

func (t ToolsConfig) BraveSearchKey() string {
	if k := strings.TrimSpace(t.BraveSearchAPIKey); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("BRAVE_API_KEY"))
}
