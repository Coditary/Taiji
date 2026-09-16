package rag

import (
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// Options configures service construction.
type Options struct {
	Local     bool
	LocalOnly bool
}

// NewService builds a scoped RAG service from Taiji config.
func NewService(cfg *taijicfg.Config, opts Options) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	ragCfg := cfg.ResolvedRAG()
	if !ragCfg.EnabledOrDefault() {
		return nil, fmt.Errorf("rag is disabled in config")
	}
	backendName := strings.ToLower(strings.TrimSpace(ragCfg.Backend))
	if backendName == "" {
		backendName = "wuji"
	}
	var backend Backend
	switch backendName {
	case "wuji":
		backend = newWujiBackend(cfg)
	default:
		return nil, fmt.Errorf("unsupported rag backend %q (supported: wuji)", ragCfg.Backend)
	}
	return &Service{
		Backend:           backend,
		Scope:             ScopeFromConfig(cfg, ScopeOptions{Local: opts.Local, LocalOnly: opts.LocalOnly}),
		DriverID:          ragCfg.Driver,
		DefaultCollection: ragCfg.DefaultCollection,
	}, nil
}

// Close releases backend resources when supported.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	if c, ok := s.Backend.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}

// DescribeScope returns the active read/write scope for debugging.
func (s *Service) DescribeScope() string {
	if s == nil {
		return ""
	}
	return s.Scope.Describe()
}
