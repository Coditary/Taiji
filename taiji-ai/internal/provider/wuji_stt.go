package provider

import (
	"context"
	"encoding/json"

	wujidriver "github.com/coditary/wuji-core/pkg/driver"
)

// ListDrivers returns installed Wuji driver metadata.
func (c *WujiCoreClient) ListDrivers(ctx context.Context) ([]wujidriver.Info, error) {
	raw, err := c.call(ctx, "ListDrivers", map[string]any{})
	if err != nil {
		return nil, err
	}
	var out []wujidriver.Info
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// EnsureDriver starts or attaches a plugin driver.
func (c *WujiCoreClient) EnsureDriver(ctx context.Context, driverID string) error {
	_, err := c.call(ctx, "EnsureDriver", map[string]any{"driver_id": driverID})
	return err
}

// GenerateText runs a non-streaming text request (including audio transcription).
func (c *WujiCoreClient) GenerateText(ctx context.Context, driverID string, req wujidriver.TextRequest) (*wujidriver.TextResponse, error) {
	raw, err := c.call(ctx, "GenerateText", map[string]any{
		"driver_id": driverID,
		"request":   req,
	})
	if err != nil {
		return nil, err
	}
	var out wujidriver.TextResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
