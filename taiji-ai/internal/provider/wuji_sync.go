package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// SyncResult describes a Wuji provider refresh attempt.
type SyncResult struct {
	Connected bool
	Endpoint  string
	Added     int
	Updated   int
	Saved     bool
	Message   string
}

// SyncProvidersFromWuji refreshes the catalog when wuji.auto_sync is enabled.
func SyncProvidersFromWuji(ctx context.Context, cfg *taijicfg.Config) (SyncResult, error) {
	if cfg == nil || !cfg.Wuji.AutoSync {
		return SyncResult{Message: "wuji auto_sync disabled (run: taiji providers refresh)"}, nil
	}
	res, err := RefreshProvidersFromWuji(ctx, cfg)
	if err != nil {
		return res, err
	}
	if res.Connected && (res.Added > 0 || res.Updated > 0) {
		if err := cfg.Save(); err != nil {
			return res, err
		}
		res.Saved = true
	}
	return res, nil
}

// RefreshProvidersFromWuji imports the Wuji provider catalog into cfg.Providers and persists when changed.
func RefreshProvidersFromWuji(ctx context.Context, cfg *taijicfg.Config) (SyncResult, error) {
	result := SyncResult{}
	if cfg == nil {
		return result, fmt.Errorf("config is nil")
	}

	endpoint := ResolveTaijiWujiEndpoint(cfg.Root, cfg.Wuji.Endpoint, cfg.Wuji.WujiRoot)
	result.Endpoint = endpoint

	root := resolveWujiRoot(cfg.Wuji.WujiRoot, cfg.Root)
	client, err := DialWujiCore(ctx, endpoint, root)
	if err != nil {
		result.Message = fmt.Sprintf("wuji unavailable at %s: %v", endpoint, err)
		return result, nil
	}
	defer client.Close()

	catalog, err := client.listProviders(ctx)
	if err != nil {
		result.Message = fmt.Sprintf("wuji ListProviders failed: %v", err)
		return result, nil
	}

	result.Connected = true
	if cfg.Providers == nil {
		cfg.Providers = map[string]taijicfg.ProviderConfig{}
	}

	for id, summary := range catalog.Providers {
		existing, hasExisting := cfg.Providers[id]
		if hasExisting && existing.Pinned {
			continue
		}
		driverID := strings.TrimSpace(summary.Driver)
		if driverID == "" {
			driverID = id
		}
		next := taijicfg.ProviderConfig{
			Type:       "wuji",
			Endpoint:   endpoint,
			WujiRoot:   strings.TrimSpace(cfg.Wuji.WujiRoot),
			ProviderID: driverID,
			Model:      summary.Model,
			Synced:     true,
			Pinned:     existing.Pinned,
		}
		if !hasExisting {
			cfg.Providers[id] = next
			result.Added++
			continue
		}
		if providerConfigChanged(existing, next) {
			cfg.Providers[id] = next
			result.Updated++
		}
	}

	if strings.TrimSpace(cfg.DefaultProvider) == "" && strings.TrimSpace(catalog.DefaultProvider) != "" {
		if _, ok := cfg.Providers[catalog.DefaultProvider]; ok {
			cfg.DefaultProvider = catalog.DefaultProvider
		}
	}

	switch {
	case len(catalog.Providers) == 0:
		result.Message = "connected to wuji (no text providers available)"
	case result.Added == 0 && result.Updated == 0:
		result.Message = fmt.Sprintf("catalog up to date (%d provider(s))", len(catalog.Providers))
	default:
		result.Message = fmt.Sprintf("refreshed catalog: %d added, %d updated", result.Added, result.Updated)
	}
	return result, nil
}

func providerConfigChanged(before, after taijicfg.ProviderConfig) bool {
	return before.Type != after.Type ||
		before.ProviderID != after.ProviderID ||
		before.Model != after.Model ||
		before.Endpoint != after.Endpoint ||
		before.WujiRoot != after.WujiRoot
}

func resolveTaijiWujiEndpoint(cfg *taijicfg.Config) string {
	if cfg == nil {
		return ResolveTaijiWujiEndpoint("", "", "")
	}
	return ResolveTaijiWujiEndpoint(cfg.Root, cfg.Wuji.Endpoint, cfg.Wuji.WujiRoot)
}

// LogSyncResult prints refresh status when useful.
func LogSyncResult(res SyncResult) {
	if res.Message == "" {
		return
	}
	if res.Connected {
		fmt.Fprintf(os.Stderr, "taiji: %s\n", res.Message)
		return
	}
	fmt.Fprintf(os.Stderr, "taiji: %s (using saved providers)\n", res.Message)
}
