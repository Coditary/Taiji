package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/wuji-core/pkg/catalog"
)

// ModelDetailsPayload is catalog metadata for the settings model inspector.
type ModelDetailsPayload struct {
	ProviderID   string `json:"provider_id"`
	ModelID      string `json:"model_id"`
	Name         string `json:"name"`
	CatalogKnown bool   `json:"catalog_known"`
	Reasoning    bool   `json:"reasoning"`
	ToolCall     bool   `json:"tool_call"`
	Context      int    `json:"context"`
	Output       int    `json:"output"`
	OpenWeights  bool   `json:"open_weights"`
	Family       string `json:"family,omitempty"`
	Lab          string `json:"lab,omitempty"`

	InputModalities  []string `json:"input_modalities,omitempty"`
	OutputModalities []string `json:"output_modalities,omitempty"`
	Attachment       bool     `json:"attachment,omitempty"`
	StructuredOutput bool     `json:"structured_output,omitempty"`
	Temperature      bool     `json:"temperature,omitempty"`
	Knowledge        string   `json:"knowledge,omitempty"`
	ReleaseDate      string   `json:"release_date,omitempty"`
	Status           string   `json:"status,omitempty"`
	CostInputUSD     float64  `json:"cost_input_usd,omitempty"`
	CostOutputUSD    float64  `json:"cost_output_usd,omitempty"`

	Notes []string `json:"notes,omitempty"`
	Lines []string `json:"lines"`
}

// GetProviderModelDetails loads Wuji catalog metadata for a provider model.
func GetProviderModelDetails(ctx context.Context, cfg *taijicfg.Config, providerID, modelID string) (ModelDetailsPayload, error) {
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ModelDetailsPayload{}, fmt.Errorf("model id is required")
	}
	if cfg == nil {
		return ModelDetailsPayload{}, fmt.Errorf("config is nil")
	}

	var details catalog.ModelDetails
	var loaded bool
	endpoint := resolveTaijiWujiEndpoint(cfg)
	root := resolveWujiRoot(cfg.Wuji.WujiRoot, cfg.Root)
	client, err := DialWujiCore(ctx, endpoint, root)
	if err == nil {
		defer client.Close()
		if rpcDetails, rpcErr := client.getProviderModelDetails(ctx, providerID, modelID); rpcErr == nil {
			details = rpcDetails
			loaded = true
		}
	}
	if !loaded {
		wujiRoot := ResolveWujiRoot(cfg)
		details, err = catalog.ModelDetailsForProvider(wujiRoot, providerID, modelID)
		if err != nil {
			return ModelDetailsPayload{}, err
		}
	}

	out := payloadFromCatalogDetails(details)
	out.Lines = FormatModelDetailLines(out)
	return out, nil
}

func payloadFromCatalogDetails(d catalog.ModelDetails) ModelDetailsPayload {
	return ModelDetailsPayload{
		ProviderID:       d.ProviderID,
		ModelID:          d.ModelID,
		Name:             d.Name,
		CatalogKnown:     d.CatalogKnown,
		Reasoning:        d.Reasoning,
		ToolCall:         d.ToolCall,
		Context:          d.Context,
		Output:           d.Output,
		OpenWeights:      d.OpenWeights,
		Family:           d.Family,
		Lab:              d.Lab,
		InputModalities:  append([]string(nil), d.InputModalities...),
		OutputModalities: append([]string(nil), d.OutputModalities...),
		Attachment:       d.Attachment,
		StructuredOutput: d.StructuredOutput,
		Temperature:      d.Temperature,
		Knowledge:        d.Knowledge,
		ReleaseDate:      d.ReleaseDate,
		Status:           d.Status,
		CostInputUSD:     d.CostInputUSD,
		CostOutputUSD:    d.CostOutputUSD,
		Notes:            append([]string(nil), d.Notes...),
	}
}

func (c *WujiCoreClient) getProviderModelDetails(ctx context.Context, providerID, modelID string) (catalog.ModelDetails, error) {
	out, err := c.call(ctx, "GetProviderModelDetails", map[string]string{
		"provider_id": providerID,
		"model_id":    modelID,
	})
	if err != nil {
		return catalog.ModelDetails{}, err
	}
	var details catalog.ModelDetails
	if err := json.Unmarshal(out, &details); err != nil {
		return catalog.ModelDetails{}, fmt.Errorf("decode GetProviderModelDetails: %w", err)
	}
	return details, nil
}

func hasModality(modalities []string, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, m := range modalities {
		if strings.EqualFold(strings.TrimSpace(m), name) {
			return true
		}
	}
	return false
}

func modalityLine(known bool, modalities []string, kind string) string {
	if !known {
		return "unknown"
	}
	if hasModality(modalities, kind) {
		return "yes"
	}
	if len(modalities) == 0 {
		return "unknown"
	}
	return "no"
}

// FormatModelDetailLines renders human-readable detail text for the TUI.
func FormatModelDetailLines(d ModelDetailsPayload) []string {
	yesNo := func(v bool) string {
		if v {
			return "yes"
		}
		return "no"
	}
	flag := func(known bool, v bool) string {
		if !known {
			return "unknown"
		}
		return yesNo(v)
	}
	tokenStr := func(n int) string {
		if n <= 0 {
			return "—"
		}
		if n >= 1_000_000 {
			return fmt.Sprintf("%.1fM tokens", float64(n)/1_000_000)
		}
		if n >= 10_000 {
			return fmt.Sprintf("%dk tokens", n/1000)
		}
		return fmt.Sprintf("%d tokens", n)
	}
	costStr := func(v float64) string {
		if v <= 0 {
			return ""
		}
		return fmt.Sprintf("$%.2f / 1M tokens", v)
	}

	var lines []string
	lines = append(lines, strings.TrimSpace(d.Name))
	lines = append(lines, d.ModelID)
	if d.ProviderID != "" {
		lines = append(lines, "Provider: "+d.ProviderID)
	}
	if d.Status != "" {
		lines = append(lines, "Status: "+d.Status)
	}
	lines = append(lines, "")
	lines = append(lines, "Capabilities (models.dev)")
	lines = append(lines, "  Tool calls        "+flag(d.CatalogKnown, d.ToolCall))
	lines = append(lines, "  Reasoning         "+flag(d.CatalogKnown, d.Reasoning))
	lines = append(lines, "  Structured output "+flag(d.CatalogKnown, d.StructuredOutput))
	lines = append(lines, "  Attachments       "+flag(d.CatalogKnown, d.Attachment))
	lines = append(lines, "  Temperature ctrl  "+flag(d.CatalogKnown, d.Temperature))
	ctxLine := tokenStr(d.Context)
	if !d.CatalogKnown || d.Context <= 0 {
		ctxLine = "unknown"
	}
	outLine := tokenStr(d.Output)
	if !d.CatalogKnown || d.Output <= 0 {
		outLine = "unknown"
	}
	lines = append(lines, "  Context window    "+ctxLine)
	lines = append(lines, "  Max output        "+outLine)
	lines = append(lines, "  Open weights      "+flag(d.CatalogKnown, d.OpenWeights))
	lines = append(lines, "  Vision / images   "+modalityLine(d.CatalogKnown, d.InputModalities, "image"))
	lines = append(lines, "  Audio             "+modalityLine(d.CatalogKnown, d.InputModalities, "audio"))
	if d.CatalogKnown && hasModality(d.OutputModalities, "audio") {
		lines[len(lines)-1] = "  Audio             yes"
	}
	lines = append(lines, "  Video             "+modalityLine(d.CatalogKnown, d.InputModalities, "video"))
	if d.CatalogKnown {
		in := strings.Join(d.InputModalities, ", ")
		out := strings.Join(d.OutputModalities, ", ")
		if in != "" {
			lines = append(lines, "  Input modalities  "+in)
		}
		if out != "" {
			lines = append(lines, "  Output modalities "+out)
		}
	}
	if inCost := costStr(d.CostInputUSD); inCost != "" {
		lines = append(lines, "  Cost (input)      "+inCost)
	}
	if outCost := costStr(d.CostOutputUSD); outCost != "" {
		lines = append(lines, "  Cost (output)     "+outCost)
	}
	if d.Family != "" {
		lines = append(lines, "")
		lines = append(lines, "Family: "+d.Family)
	}
	if d.Lab != "" {
		lines = append(lines, "Lab: "+d.Lab)
	}
	if d.Knowledge != "" {
		lines = append(lines, "Knowledge cutoff: "+d.Knowledge)
	}
	if d.ReleaseDate != "" {
		lines = append(lines, "Release: "+d.ReleaseDate)
	}
	if len(d.Notes) > 0 {
		lines = append(lines, "")
		for _, note := range d.Notes {
			note = strings.TrimSpace(note)
			if note != "" {
				lines = append(lines, note)
			}
		}
	}
	return lines
}
