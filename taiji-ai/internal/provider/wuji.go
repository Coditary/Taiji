package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/modelref"
	"github.com/coditary/taiji/internal/wujipath"
	"github.com/coditary/wuji-core/chatformat"
)

type wujiProvider struct {
	id          string
	driverID    string
	endpoint    string
	wujiRoot    string
	model       string
	maxTokens   int
	temperature float64
	client      *WujiCoreClient
}

func newWujiProvider(id string, pc taijicfg.ProviderConfig, cfg *taijicfg.Config) (Provider, error) {
	driverID := strings.TrimSpace(pc.ProviderID)
	if driverID == "" {
		driverID = id
	}
	cfgRoot := ""
	wujiRoot := strings.TrimSpace(pc.WujiRoot)
	if cfg != nil {
		cfgRoot = cfg.Root
		if wujiRoot == "" {
			wujiRoot = strings.TrimSpace(cfg.Wuji.WujiRoot)
		}
	}
	return &wujiProvider{
		id:          id,
		driverID:    driverID,
		endpoint:    resolveWujiEndpoint(pc, cfg),
		wujiRoot:    resolveWujiRoot(wujiRoot, cfgRoot),
		model:       strings.TrimSpace(pc.Model),
		maxTokens:   pc.MaxTokens,
		temperature: pc.Temperature,
	}, nil
}

func (p *wujiProvider) ensureClient(ctx context.Context) error {
	if p.client != nil {
		return nil
	}
	client, err := DialWujiCore(ctx, p.endpoint, p.wujiRoot)
	if err != nil {
		return fmt.Errorf("connect wuji core at %s: %w", p.endpoint, err)
	}
	p.client = client
	return nil
}

func resolveWujiEndpoint(pc taijicfg.ProviderConfig, cfg *taijicfg.Config) string {
	cfgRoot := ""
	if cfg != nil {
		cfgRoot = cfg.Root
	}
	ep := strings.TrimSpace(pc.Endpoint)
	if ep != "" && ep != "auto" {
		return normalizeWujiEndpoint(ep, cfgRoot)
	}
	wujiRoot := strings.TrimSpace(pc.WujiRoot)
	if wujiRoot == "" && cfg != nil {
		wujiRoot = strings.TrimSpace(cfg.Wuji.WujiRoot)
	}
	return defaultWujiCoreEndpoint(resolveWujiRoot(wujiRoot, cfgRoot))
}

func (p *wujiProvider) ID() string   { return p.id }
func (p *wujiProvider) Type() string { return "wuji" }

func isTransientWujiErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection is closing") ||
		strings.Contains(msg, "Unavailable") ||
		strings.Contains(msg, "transport:")
}

func (p *wujiProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	messages := req.Messages
	if len(req.Tools) > 0 {
		messages = fromChatformatMessages(chatformat.SanitizeMessages(toChatformatMessages(messages)))
	}
	model := firstNonEmpty(req.Model, p.model)
	model = modelref.Expand(model, modelref.Candidates(nil, p.id, p.model, model))
	reqBody := map[string]any{
		"messages":    messages,
		"tools":       req.Tools,
		"model":       model,
		"max_tokens":  firstPositive(req.MaxTokens, p.maxTokens),
		"temperature": firstPositiveFloat(req.Temperature, p.temperature),
		"stream":      req.OnDelta != nil,
	}
	if req.ContextWindow > 0 {
		reqBody["context_window"] = req.ContextWindow
	}
	if tc := strings.TrimSpace(req.ToolChoice); tc != "" {
		reqBody["tool_choice"] = tc
	}
	payload := map[string]any{
		"driver_id": p.driverID,
		"request":   reqBody,
	}
	var out []byte
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		if err := p.ensureClient(ctx); err != nil {
			return nil, err
		}
		if req.OnDelta != nil {
			out, err = p.client.generateTextStream(ctx, payload, req.OnDelta, req.OnThinkDelta)
		} else {
			out, err = p.client.call(ctx, "GenerateText", payload)
		}
		if err == nil {
			break
		}
		if !isTransientWujiErr(err) || attempt == 1 {
			return nil, err
		}
		if p.client != nil {
			_ = p.client.Close()
			p.client = nil
		}
		// wuji-core can keep a dead gRPC link to llama after idle driver shutdown;
		// reconnecting only our client is not enough — restart the core daemon.
		_ = wujipath.RestartCoreDaemon(ctx, p.wujiRoot, p.endpoint)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
	if err != nil {
		return nil, err
	}
	var wujiResp struct {
		Text             string          `json:"text"`
		ThinkingBlocks   []ThinkingBlock `json:"thinking_blocks"`
		Thinking         string          `json:"thinking"`
		TokensUsed       int             `json:"tokens_used"`
		InputTokens      int             `json:"input_tokens"`
		LatencyMs        int64           `json:"latency_ms"`
		TokensPerSec     float64         `json:"tokens_per_sec"`
		EstimatedCostUSD float64         `json:"estimated_cost_usd"`
		FinishReason     string          `json:"finish_reason"`
		ToolCalls        []ToolCall      `json:"tool_calls"`
	}
	if err := json.Unmarshal(out, &wujiResp); err != nil {
		return nil, fmt.Errorf("decode wuji GenerateText response: %w", err)
	}
	blocks := wujiResp.ThinkingBlocks
	if len(blocks) == 0 && strings.TrimSpace(wujiResp.Thinking) != "" {
		blocks = []ThinkingBlock{{Index: 0, Text: wujiResp.Thinking}}
	}
	return &CompletionResponse{
		Content:        wujiResp.Text,
		ThinkingBlocks: blocks,
		ToolCalls:      wujiResp.ToolCalls,
		FinishReason:   wujiResp.FinishReason,
		Usage: Usage{
			InputTokens:      wujiResp.InputTokens,
			OutputTokens:     wujiResp.TokensUsed,
			LatencyMs:        wujiResp.LatencyMs,
			TokensPerSec:     wujiResp.TokensPerSec,
			EstimatedCostUSD: wujiResp.EstimatedCostUSD,
		},
	}, nil
}

func firstPositive(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}

func firstPositiveFloat(a, b float64) float64 {
	if a > 0 {
		return a
	}
	return b
}

func firstNonEmpty(parts ...string) string {
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			return p
		}
	}
	return ""
}
