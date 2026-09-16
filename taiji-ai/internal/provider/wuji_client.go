package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	taijicfg "github.com/coditary/taiji/internal/config"
	wujiv1 "github.com/coditary/wuji/api/proto/v1"
)

// WujiCoreClient talks to a Wuji core daemon over gRPC.
type WujiCoreClient struct {
	conn   *grpc.ClientConn
	client wujiv1.WujiCoreClient
}

// DialWujiCore connects to the Wuji core at endpoint and verifies with Ping.
func DialWujiCore(ctx context.Context, endpoint string) (*WujiCoreClient, error) {
	target := dialTarget(endpoint)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}
	c := wujiv1.NewWujiCoreClient(conn)
	if _, err := c.Ping(ctx, &wujiv1.PingRequest{}); err != nil {
		_ = conn.Close()
		return nil, FormatWujiConnectError(endpoint, fmt.Errorf("ping: %w", err))
	}
	return &WujiCoreClient{conn: conn, client: c}, nil
}

func dialTarget(endpoint string) string {
	ep := strings.TrimSpace(endpoint)
	if strings.HasPrefix(ep, "unix://") {
		return "unix:" + strings.TrimPrefix(ep, "unix://")
	}
	return ep
}

// Close closes the underlying gRPC connection.
func (c *WujiCoreClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *WujiCoreClient) call(ctx context.Context, method string, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Call(ctx, &wujiv1.CallRequest{Method: method, PayloadJson: raw})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != "" {
		return nil, fmt.Errorf("%s", resp.GetError())
	}
	return resp.GetPayloadJson(), nil
}

// RunRAG executes a RAG task through Wuji core and returns the WDD shape JSON document.
func (c *WujiCoreClient) RunRAG(ctx context.Context, driverID string, request map[string]any) (json.RawMessage, error) {
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		driverID = taijicfg.DefaultRAGConfig().Driver
	}
	out, err := c.call(ctx, "RunRAG", map[string]any{
		"driver_id": driverID,
		"request":   request,
	})
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Shape json.RawMessage `json:"shape"`
	}
	if err := json.Unmarshal(out, &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Shape) == 0 {
		return nil, fmt.Errorf("empty rag response")
	}
	return wrapper.Shape, nil
}

func (c *WujiCoreClient) generateTextStream(ctx context.Context, payload map[string]any, onDelta, onThinkDelta func(string) error) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	stream, err := c.client.GenerateTextStream(ctx, &wujiv1.CallRequest{PayloadJson: raw})
	if err != nil {
		return nil, err
	}
	for {
		chunk, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		if chunk.GetError() != "" {
			return nil, fmt.Errorf("%s", chunk.GetError())
		}
		if delta := chunk.GetDelta(); delta != "" && onDelta != nil {
			if err := onDelta(delta); err != nil {
				return nil, err
			}
		}
		if think := chunk.GetThinkDelta(); think != "" && onThinkDelta != nil {
			if err := onThinkDelta(think); err != nil {
				return nil, err
			}
		}
		if chunk.GetDone() {
			return chunk.GetPayloadJson(), nil
		}
	}
}

// ResolveTaijiWujiEndpoint returns the Wuji core address from Taiji project config.
func ResolveTaijiWujiEndpoint(cfgRoot, wujiEndpoint, wujiRoot string) string {
	ep := strings.TrimSpace(wujiEndpoint)
	if ep != "" && ep != "auto" {
		return normalizeWujiEndpoint(ep, cfgRoot)
	}
	return defaultWujiCoreEndpoint(resolveWujiRoot(strings.TrimSpace(wujiRoot), cfgRoot))
}
