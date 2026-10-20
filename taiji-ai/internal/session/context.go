package session

import (
	"strings"

	"github.com/coditary/taiji/internal/config"
)

const defaultContextWindow = 128_000

// ContextSnapshot reports session usage for UI display.
type ContextSnapshot struct {
	ContextTokens    int
	ContextWindow    int
	ContextUsedPct   float64
	InputTokens      int
	OutputTokens     int
	EstimatedCostUSD float64
}

// ContextSnapshot builds live context and cost stats for a session.
func (s *Session) ContextSnapshot(cfg *config.Config, providerID string) ContextSnapshot {
	if s == nil {
		return ContextSnapshot{ContextWindow: defaultContextWindow}
	}

	st := s.State.Stats
	contextTokens := st.LastInputTokens
	if contextTokens <= 0 {
		contextTokens = EstimateMessagesTokens(s.State.Messages)
		if contextTokens > 0 {
			contextTokens += 512 // rough system/tools overhead
		}
	}

	window := s.resolveContextWindow(cfg, providerID)
	pct := 0.0
	if window > 0 && contextTokens > 0 {
		pct = float64(contextTokens) / float64(window) * 100
		if pct > 100 {
			pct = 100
		}
	}

	return ContextSnapshot{
		ContextTokens:    contextTokens,
		ContextWindow:    window,
		ContextUsedPct:   pct,
		InputTokens:      st.InputTokens,
		OutputTokens:     st.OutputTokens,
		EstimatedCostUSD: st.EstimatedCostUSD,
	}
}

func (s *Session) resolveContextWindow(cfg *config.Config, providerID string) int {
	if s != nil {
		if v := s.State.Settings.ContextWindow; v != nil && *v > 0 {
			return *v
		}
	}
	return ResolveContextWindow(cfg, providerID)
}

// ResolveContextWindow returns the configured or inferred context limit for a provider.
func ResolveContextWindow(cfg *config.Config, providerID string) int {
	providerID = strings.TrimSpace(providerID)
	model := ""
	if cfg != nil {
		if pc, ok := cfg.Providers[providerID]; ok {
			if pc.ContextWindow > 0 {
				return pc.ContextWindow
			}
			model = strings.TrimSpace(pc.Model)
		}
		if w := contextWindowForModel(model); w > 0 {
			return w
		}
		if cfg.Agent.DefaultContextWindow > 0 {
			return cfg.Agent.DefaultContextWindow
		}
	}
	if w := contextWindowForModel(model); w > 0 {
		return w
	}
	return defaultContextWindow
}
