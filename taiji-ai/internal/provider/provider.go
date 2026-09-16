package provider

import (
	"context"
	"strings"
)

// Role identifies a chat message author.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is a chat message passed to providers.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolDef describes a callable tool for the model.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// CompletionRequest is a provider-agnostic chat completion request.
type CompletionRequest struct {
	Messages      []Message
	Tools         []ToolDef
	Model         string
	MaxTokens     int
	Temperature   float64
	ContextWindow int
	ToolChoice    string // "auto", "required", or "none"
	OnDelta       func(delta string) error
	OnThinkDelta  func(delta string) error
}

// ToolCall is a model-requested tool invocation.
type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"` // JSON object string
}

// Usage reports token usage when available.
type Usage struct {
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	LatencyMs        int64   `json:"latency_ms,omitempty"`
	TokensPerSec     float64 `json:"tokens_per_sec,omitempty"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
}

// ThinkingBlock is one reasoning segment from the model.
type ThinkingBlock struct {
	Index  int    `json:"index"`
	Text   string `json:"text"`
	Source string `json:"source,omitempty"`
}

// JoinThinkingBlocks concatenates block text.
func JoinThinkingBlocks(blocks []ThinkingBlock) string {
	if len(blocks) == 0 {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if s := strings.TrimSpace(b.Text); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n\n")
}

// CompletionResponse is a provider-agnostic chat completion response.
type CompletionResponse struct {
	Content        string
	ThinkingBlocks []ThinkingBlock
	ToolCalls      []ToolCall
	FinishReason   string
	Usage          Usage
}

// Provider completes chat requests. Implementations are selected via YAML `type`.
type Provider interface {
	ID() string
	Type() string
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}
