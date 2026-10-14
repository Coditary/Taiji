package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	wujidriver "github.com/coditary/wuji-core/pkg/driver"
)

// WujiImageResponse is the decoded GenerateImage RPC payload.
type WujiImageResponse struct {
	Path   string   `json:"path"`
	Paths  []string `json:"paths"`
	Format string   `json:"format"`
}

// GenerateImage runs text-to-image (or other image tasks) through Wuji core.
// Leave driverID empty so Wuji picks the configured image capability driver.
func (c *WujiCoreClient) GenerateImage(ctx context.Context, driverID string, req wujidriver.ImageRequest) (*WujiImageResponse, error) {
	out, err := c.call(ctx, "GenerateImage", map[string]any{
		"driver_id": strings.TrimSpace(driverID),
		"request":   req,
	})
	if err != nil {
		return nil, err
	}
	var resp WujiImageResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("decode GenerateImage: %w", err)
	}
	if strings.TrimSpace(resp.Path) == "" && len(resp.Paths) == 0 {
		return nil, fmt.Errorf("empty image response")
	}
	return &resp, nil
}
