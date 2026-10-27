package provider

import (
	"context"
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// dummyProvider is a deterministic test provider (no Wuji or network).
// YAML: type: dummy, model: alpha|beta (default alpha).
type dummyProvider struct {
	id     string
	preset string
}

func newDummyProvider(id string, pc taijicfg.ProviderConfig) (Provider, error) {
	preset := strings.ToLower(strings.TrimSpace(pc.Model))
	if preset == "" {
		preset = "alpha"
	}
	switch preset {
	case "alpha", "beta":
	default:
		return nil, fmt.Errorf("dummy provider %q: unknown model preset %q (use alpha or beta)", id, pc.Model)
	}
	return &dummyProvider{id: id, preset: preset}, nil
}

func (p *dummyProvider) ID() string   { return p.id }
func (p *dummyProvider) Type() string { return "dummy" }

func (p *dummyProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	user := lastUserMessageContent(req.Messages)
	text := dummyReply(p.preset, user)
	if req.OnDelta != nil && text != "" {
		if err := req.OnDelta(text); err != nil {
			return nil, err
		}
	}
	return &CompletionResponse{Content: text, FinishReason: "stop"}, nil
}

func lastUserMessageContent(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

// dummyReply maps user text to assistant text for integration tests without Wuji.
func dummyReply(preset, user string) string {
	switch preset {
	case "beta":
		switch user {
		case "Hallo":
			return "Servus"
		case "Ping":
			return "Bing"
		default:
			if user == "" {
				return ""
			}
			return "beta:" + user
		}
	default: // alpha
		switch user {
		case "Ping":
			return "Pong"
		default:
			return user
		}
	}
}
