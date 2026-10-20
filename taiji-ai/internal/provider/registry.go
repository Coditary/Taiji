package provider

import (
	"context"
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// Registry holds named providers loaded from config.
type Registry struct {
	defaultID string
	providers map[string]Provider
}

// NewRegistry builds providers from config entries, syncing from Wuji first when enabled.
func NewRegistry(ctx context.Context, cfg *taijicfg.Config, quiet bool) (*Registry, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	syncRes, err := SyncProvidersFromWuji(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if !quiet {
		LogSyncResult(syncRes)
	}

	providers := make(map[string]Provider, len(cfg.Providers))
	for id, pc := range cfg.Providers {
		p, err := buildProvider(id, pc, cfg)
		if err != nil {
			return nil, fmt.Errorf("provider %q: %w", id, err)
		}
		providers[id] = p
	}
	defaultID := strings.TrimSpace(cfg.DefaultProvider)
	if defaultID == "" && len(providers) == 1 {
		for id := range providers {
			defaultID = id
		}
	}
	return &Registry{defaultID: defaultID, providers: providers}, nil
}

// NewStaticRegistry wires a fixed provider map for tests and tooling.
func NewStaticRegistry(defaultID string, providers map[string]Provider) *Registry {
	return &Registry{defaultID: defaultID, providers: providers}
}

func buildProvider(id string, pc taijicfg.ProviderConfig, cfg *taijicfg.Config) (Provider, error) {
	typ := strings.ToLower(strings.TrimSpace(pc.Type))
	switch typ {
	case "wuji":
		return newWujiProvider(id, pc, cfg)
	case "openai":
		return newOpenAIProvider(id, pc)
	case "anthropic":
		return newAnthropicProvider(id, pc)
	default:
		return nil, fmt.Errorf("unknown provider type %q (supported: wuji, openai, anthropic)", pc.Type)
	}
}

// Default returns the configured default provider.
func (r *Registry) Default() (Provider, error) {
	if r.defaultID == "" {
		return nil, fmt.Errorf("no default_provider configured")
	}
	return r.Get(r.defaultID)
}

// Get returns a provider by id.
func (r *Registry) Get(id string) (Provider, error) {
	p, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", id)
	}
	return p, nil
}

// List returns all provider ids and types.
func (r *Registry) List() []Info {
	out := make([]Info, 0, len(r.providers))
	for id, p := range r.providers {
		out = append(out, Info{ID: id, Type: p.Type()})
	}
	return out
}

// Info summarizes a registered provider.
type Info struct {
	ID   string
	Type string
}
