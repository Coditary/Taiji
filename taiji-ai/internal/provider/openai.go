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

type openAIProvider struct {
	id          string
	baseURL     string
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
}

func newOpenAIProvider(id string, pc taijicfg.ProviderConfig) (Provider, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(pc.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
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
	return &openAIProvider{
		id:          id,
		baseURL:     baseURL,
		apiKey:      apiKey,
		model:       model,
		maxTokens:   pc.MaxTokens,
		temperature: pc.Temperature,
		client:      &http.Client{Timeout: 10 * time.Minute},
	}, nil
}

func (p *openAIProvider) ID() string   { return p.id }
func (p *openAIProvider) Type() string { return "openai" }

func (p *openAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	model := p.model
	if strings.TrimSpace(req.Model) != "" {
		model = req.Model
	}
	cfMsgs := toChatformatMessages(req.Messages)
	cfTools := toChatformatTools(req.Tools)
	body := map[string]any{
		"model":    model,
		"messages": chatformat.MessagesOpenAI(cfMsgs),
	}
	if len(cfTools) > 0 {
		body["tools"] = chatformat.ToolsOpenAI(cfTools)
		if tc := strings.TrimSpace(req.ToolChoice); tc != "" {
			body["tool_choice"] = chatformat.OpenAIToolChoice(parseToolChoice(tc), "")
		}
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	} else if p.maxTokens > 0 {
		body["max_tokens"] = p.maxTokens
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
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		return nil, fmt.Errorf("openai api %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return parseOpenAIResponse(respBody)
}

func parseOpenAIResponse(raw []byte) (*CompletionResponse, error) {
	var parsed struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai: empty choices")
	}
	choice := parsed.Choices[0]
	resp := &CompletionResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			InputTokens:  parsed.Usage.PromptTokens,
			OutputTokens: parsed.Usage.CompletionTokens,
		},
	}
	for _, tc := range choice.Message.ToolCalls {
		resp.ToolCalls = append(resp.ToolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: tc.Function.Arguments,
		})
	}
	return resp, nil
}
