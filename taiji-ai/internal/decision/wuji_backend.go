package decision

import (
	"context"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

type wujiBackend struct {
	endpoint string
	wujiRoot string
	driverID string
	model    string
	client   *provider.WujiCoreClient
}

func newWujiBackend(cfg *taijicfg.Config, driverID, model string) *wujiBackend {
	return &wujiBackend{
		endpoint: provider.ResolveTaijiWujiEndpoint(cfg.Root, cfg.Wuji.Endpoint, cfg.Wuji.WujiRoot),
		wujiRoot: provider.ResolveWujiRoot(cfg),
		driverID: strings.TrimSpace(driverID),
		model:    strings.TrimSpace(model),
	}
}

func (b *wujiBackend) Decide(ctx context.Context, req Request) (*Response, error) {
	client, err := b.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = b.model
	}
	raw, err := client.ProduceDecision(ctx, b.driverID, provider.DecisionRequest{
		Text:      req.Text,
		Model:     model,
		Questions: toWujiQuestions(req.Questions),
	})
	if err != nil {
		return nil, err
	}
	return ParseResponse(raw, b.driverID, "wuji")
}

func (b *wujiBackend) clientFor(ctx context.Context) (*provider.WujiCoreClient, error) {
	if b.client != nil {
		return b.client, nil
	}
	client, err := provider.DialWujiCore(ctx, b.endpoint, b.wujiRoot)
	if err != nil {
		return nil, err
	}
	b.client = client
	return b.client, nil
}

func (b *wujiBackend) Close() error {
	if b.client == nil {
		return nil
	}
	err := b.client.Close()
	b.client = nil
	return err
}

func toWujiQuestions(questions []Question) []provider.DecisionQuestion {
	out := make([]provider.DecisionQuestion, 0, len(questions))
	for _, q := range questions {
		out = append(out, provider.DecisionQuestion{
			ID:           q.ID,
			Type:         string(q.Type),
			Instructions: q.Instructions,
			Criteria:     append([]string(nil), q.Criteria...),
		})
	}
	return out
}
