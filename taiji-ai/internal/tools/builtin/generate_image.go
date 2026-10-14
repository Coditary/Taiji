package builtin

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
	wujidriver "github.com/coditary/wuji-core/pkg/driver"
)

const generateImageTimeout = 15 * time.Minute

// EnableImageGeneration registers generate_image when Wuji is configured.
func EnableImageGeneration(reg *tools.Registry, cfg *taijicfg.Config) {
	if reg == nil || cfg == nil {
		return
	}
	reg.Register(NewGenerateImage(cfg))
}

// GenerateImage creates images via Wuji core.
type GenerateImage struct {
	endpoint string
	wujiRoot string
	client   *provider.WujiCoreClient
}

// NewGenerateImage builds a Wuji-backed image generation tool.
func NewGenerateImage(cfg *taijicfg.Config) GenerateImage {
	return GenerateImage{
		endpoint: provider.ResolveTaijiWujiEndpoint(cfg.Root, cfg.Wuji.Endpoint, cfg.Wuji.WujiRoot),
		wujiRoot: provider.ResolveWujiRoot(cfg),
	}
}

func (GenerateImage) Name() string { return "generate_image" }

func (GenerateImage) Description() string {
	return "Generate an image from a text prompt using Wuji (local diffusion). Returns the saved file path; show it in chat with markdown ![alt](absolute-path)."
}

func (GenerateImage) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "What to generate",
			},
			"negative_prompt": map[string]any{
				"type":        "string",
				"description": "What to avoid in the image (optional)",
			},
			"width": map[string]any{
				"type":        "integer",
				"description": "Image width in pixels (optional, backend default if omitted)",
			},
			"height": map[string]any{
				"type":        "integer",
				"description": "Image height in pixels (optional)",
			},
			"steps": map[string]any{
				"type":        "integer",
				"description": "Diffusion steps (optional)",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Model or checkpoint name (optional)",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t GenerateImage) Run(ctx context.Context, args map[string]any) (string, error) {
	res, err := t.RunWithMetadata(ctx, args)
	return res.Text, err
}

func (t GenerateImage) RunWithMetadata(ctx context.Context, args map[string]any) (tools.RunResult, error) {
	prompt, _ := args["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return tools.RunResult{}, fmt.Errorf("prompt is required")
	}

	req := wujidriver.ImageRequest{
		Task:           wujidriver.ImageTaskGenerate,
		Prompt:         prompt,
		NegativePrompt: stringArg(args, "negative_prompt"),
		Model:          stringArg(args, "model"),
		Width:          intArg(args, "width"),
		Height:         intArg(args, "height"),
		Steps:          intArg(args, "steps"),
	}

	runCtx, cancel := context.WithTimeout(ctx, generateImageTimeout)
	defer cancel()

	client, err := t.clientFor(runCtx)
	if err != nil {
		return tools.RunResult{}, err
	}
	resp, err := client.GenerateImage(runCtx, "", req)
	if err != nil {
		return tools.RunResult{}, err
	}

	paths := imagePaths(resp)
	text := formatGenerateImageResult(prompt, resp)
	return tools.RunResult{
		Text: text,
		Data: map[string]any{
			"paths":  paths,
			"format": resp.Format,
		},
	}, nil
}

func (t *GenerateImage) clientFor(ctx context.Context) (*provider.WujiCoreClient, error) {
	if t.client != nil {
		return t.client, nil
	}
	client, err := provider.DialWujiCore(ctx, t.endpoint, t.wujiRoot)
	if err != nil {
		return nil, err
	}
	t.client = client
	return t.client, nil
}

func imagePaths(resp *provider.WujiImageResponse) []string {
	if resp == nil {
		return nil
	}
	if len(resp.Paths) > 0 {
		return resp.Paths
	}
	if strings.TrimSpace(resp.Path) != "" {
		return []string{resp.Path}
	}
	return nil
}

func formatGenerateImageResult(prompt string, resp *provider.WujiImageResponse) string {
	paths := imagePaths(resp)
	if len(paths) == 0 {
		return "no image path returned"
	}

	alt := strings.TrimSpace(prompt)
	if alt == "" {
		alt = "generated image"
	}
	if len(alt) > 80 {
		alt = alt[:77] + "..."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "format: %s\n", strings.TrimSpace(resp.Format))
	absPaths := make([]string, 0, len(paths))
	for _, p := range paths {
		abs := p
		if resolved, err := filepath.Abs(p); err == nil {
			abs = resolved
		}
		absPaths = append(absPaths, abs)
	}
	for i, abs := range absPaths {
		if i == 0 {
			fmt.Fprintf(&b, "path: %s\n", abs)
		} else {
			fmt.Fprintf(&b, "path[%d]: %s\n", i, abs)
		}
	}
	fmt.Fprintf(&b, "markdown: ![%s](%s)", alt, absPaths[0])
	return strings.TrimSpace(b.String())
}

func stringArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return strings.TrimSpace(v)
}

func intArg(args map[string]any, key string) int {
	switch v := args[key].(type) {
	case float64:
		if v > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	case string:
		// ignore parse errors; zero means backend default
	}
	return 0
}
