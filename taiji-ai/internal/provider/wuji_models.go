package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/wuji-core/pkg/catalog"
)

// ProviderModelOption is a selectable model for a Wuji text provider.
type ProviderModelOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type wujiProviderModelsResponse struct {
	ProviderID string                `json:"provider_id"`
	Models     []ProviderModelOption `json:"models"`
}

// ListProviderModels returns models for a provider from Wuji (core RPC, with catalog fallback).
func ListProviderModels(ctx context.Context, cfg *taijicfg.Config, providerID string) ([]ProviderModelOption, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, fmt.Errorf("provider id is required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	endpoint := resolveTaijiWujiEndpoint(cfg)
	root := resolveWujiRoot(cfg.Wuji.WujiRoot, cfg.Root)
	client, err := DialWujiCore(ctx, endpoint, root)
	if err == nil {
		defer client.Close()
		models, rpcErr := client.listProviderModels(ctx, providerID)
		if rpcErr == nil && len(models) > 0 {
			return models, nil
		}
	}

	wujiRoot := ResolveWujiRoot(cfg)
	models, err := catalog.ProviderModels(wujiRoot, providerID)
	if err != nil {
		return nil, err
	}
	out := make([]ProviderModelOption, 0, len(models))
	for _, m := range models {
		out = append(out, ProviderModelOption{ID: m.ID, Name: m.Name})
	}
	return out, nil
}

func (c *WujiCoreClient) listProviderModels(ctx context.Context, providerID string) ([]ProviderModelOption, error) {
	out, err := c.call(ctx, "ListProviderModels", map[string]string{"provider_id": providerID})
	if err != nil {
		return nil, err
	}
	var resp wujiProviderModelsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("decode ListProviderModels: %w", err)
	}
	return resp.Models, nil
}
