package provider

import (
	"context"
	"encoding/json"
	"fmt"
)

type wujiProviderCatalog struct {
	DefaultProvider string                         `json:"default_provider"`
	Providers       map[string]wujiProviderSummary `json:"providers"`
}

type wujiProviderSummary struct {
	Model  string `json:"model"`
	Driver string `json:"driver"`
	Kind   string `json:"kind"`
}

func (c *WujiCoreClient) listProviders(ctx context.Context) (*wujiProviderCatalog, error) {
	out, err := c.call(ctx, "ListProviders", map[string]any{})
	if err != nil {
		return nil, err
	}
	var catalog wujiProviderCatalog
	if err := json.Unmarshal(out, &catalog); err != nil {
		return nil, fmt.Errorf("decode ListProviders: %w", err)
	}
	if catalog.Providers == nil {
		catalog.Providers = map[string]wujiProviderSummary{}
	}
	return &catalog, nil
}
