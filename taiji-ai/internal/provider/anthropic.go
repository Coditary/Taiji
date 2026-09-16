package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/wuji/chatformat"
)

type anthropicProvider struct {
	id          string
	baseURL     string
	apiKey      string
	model       string
	version     string
	maxTokens   int
	temperature float64
	client      *http.Client
}

func newAnthropicProvider(id string, pc taijicfg.ProviderConfig) (Provider, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(pc.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	apiKey := strings.TrimSpace(pc.APIKey)
	if apiKey == "" && pc.APIKeyEnv != "" {
		apiKey = strings.TrimSpace(os.Getenv(pc.APIKeyEnv))
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api_key or api_key_env is required")
	}
	model := strings.TrimSpace(pc.Model)
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}
	version := strings.TrimSpace(pc.AnthropicVersion)
	if version == "" {
		version = "2023-06-01"
	}
	return &anthropicProvider{
		id:          id,
		baseURL:     baseURL,
		apiKey:      apiKey,
		model:       model,
		version:     version,
		maxTokens:   pc.MaxTokens,
		temperature: pc.Temperature,
		client:      &http.Client{Timeout: 10 * time.Minute},
	}, nil
}

func (p *anthropicProvider) ID() string   { return p.id }
func (p *anthropicProvider) Type() string { return "anthropic" }

func (p *anthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	model := p.model
	if strings.TrimSpace(req.Model) != "" {
		model = req.Model
	}
	cfMsgs := toChatformatMessages(req.Messages)
	cfTools := toChatformatTools(req.Tools)
	nameMaps := chatformat.NameMaps{}
	var anthropicTools []map[string]any
	if len(cfTools) > 0 {
		anthropicTools, nameMaps = chatformat.ToolsAnthropic(cfTools)
	}
	system, messages := chatformat.MessagesAnthropic(cfMsgs, nameMaps)
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = p.maxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}
	if system != "" {
		body["system"] = system
	}
	if len(anthropicTools) > 0 {
		body["tools"] = anthropicTools
		if tc := strings.TrimSpace(req.ToolChoice); tc != "" {
			body["tool_choice"] = chatformat.AnthropicToolChoice(parseToolChoice(tc), "", nameMaps)
		}
	}
	temp := p.temperature
	if req.Temperature > 0 {
		temp = req.Temperature
	}
	if temp > 0 {
		body["temperature"] = temp
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", p.version)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("anthropic api %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return parseAnthropicResponse(respBody, nameMaps)
}

func parseAnthropicResponse(raw []byte, maps chatformat.NameMaps) (*CompletionResponse, error) {
	var parsed struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Input any    `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	resp := &CompletionResponse{
		FinishReason: parsed.StopReason,
		Usage: Usage{
			InputTokens:  parsed.Usage.InputTokens,
			OutputTokens: parsed.Usage.OutputTokens,
		},
	}
	for _, block := range parsed.Content {
		switch block.Type {
		case "text":
			resp.Content += block.Text
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:   block.ID,
				Name: chatformat.RestoreToolName(block.Name, maps),
				Args: string(args),
			})
		}
	}
	return resp, nil
}
