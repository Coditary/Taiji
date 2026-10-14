package config

import "strings"

const (
	defaultDecisionBackend = "wuji"
	defaultDecisionDriver  = "laya"
)

// DecisionConfig configures Taiji's pluggable structured decision layer.
// Backends: wuji (gRPC via wuji-core, driver selects laya/jev/…), http (laya-serve style).
type DecisionConfig struct {
	Enabled *bool  `yaml:"enabled,omitempty"`
	Backend string `yaml:"backend,omitempty"`
	Driver  string `yaml:"driver,omitempty"`
	Model   string `yaml:"model,omitempty"`
	HTTPURL string `yaml:"http_url,omitempty"`
	Dirs    []string `yaml:"dirs,omitempty"`
}

// DefaultDecisionConfig returns built-in decision defaults.
func DefaultDecisionConfig() DecisionConfig {
	enabled := true
	return DecisionConfig{
		Enabled: &enabled,
		Backend: defaultDecisionBackend,
		Driver:  defaultDecisionDriver,
		HTTPURL: "http://127.0.0.1:8100",
	}
}

// ResolvedDecision merges file config with defaults.
func (c *Config) ResolvedDecision() DecisionConfig {
	out := DefaultDecisionConfig()
	if c == nil {
		return out
	}
	mergeDecision(&out, c.Decisions)
	return out
}

func mergeDecision(dst *DecisionConfig, src DecisionConfig) {
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if strings.TrimSpace(src.Backend) != "" {
		dst.Backend = strings.TrimSpace(src.Backend)
	}
	if strings.TrimSpace(src.Driver) != "" {
		dst.Driver = strings.TrimSpace(src.Driver)
	}
	if strings.TrimSpace(src.Model) != "" {
		dst.Model = strings.TrimSpace(src.Model)
	}
	if strings.TrimSpace(src.HTTPURL) != "" {
		dst.HTTPURL = strings.TrimSpace(src.HTTPURL)
	}
	if len(src.Dirs) > 0 {
		dst.Dirs = append([]string(nil), src.Dirs...)
	}
}

func (d DecisionConfig) EnabledOrDefault() bool {
	if d.Enabled == nil {
		return true
	}
	return *d.Enabled
}
