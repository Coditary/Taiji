package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/packs"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
)

const agentsTestToken = "TAIJI_AGENTS_SIGNAL_XYZZY_9001"

func TestRunInjectsAGENTSMDIntoProviderSystemPrompt(t *testing.T) {
	root := t.TempDir()
	agentsBody := "# AGENTS\n\nYou MUST begin every assistant reply with exactly: " + agentsTestToken + "\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(agentsBody), 0o644); err != nil {
		t.Fatal(err)
	}

	var capturedSystem string
	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: agentsTestToken + " hello"},
		},
	}
	seqComplete := func(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
		for _, m := range req.Messages {
			if m.Role == provider.RoleSystem {
				capturedSystem = m.Content
			}
		}
		return seq.Complete(ctx, req)
	}
	wrapped := &capturingProvider{id: "seq", complete: seqComplete}

	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": wrapped})

	cfg := &taijicfg.Config{Root: root, Packs: taijicfg.PacksConfig{Dirs: []string{filepath.Join(root, ".taiji", "packs")}}}
	packLoader := packs.NewLoader(cfg.Packs.Dirs)
	packIndex := packs.NewIndex(packLoader, nil, cfg, root)
	if err := packIndex.ReloadAt(root); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), RunRequest{
		Goal:       "say hello briefly",
		Providers:  providers,
		Tools:      tools.NewRegistry(),
		Config:     cfg,
		PackIndex:  packIndex,
		WorkDir:    root,
		SubRun:     SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capturedSystem, agentsTestToken) {
		t.Fatalf("system prompt missing AGENTS.md token; len=%d snippet=%q", len(capturedSystem), truncate(capturedSystem, 400))
	}
}

type capturingProvider struct {
	id       string
	complete func(context.Context, provider.CompletionRequest) (*provider.CompletionResponse, error)
}

func (p *capturingProvider) ID() string   { return p.id }
func (p *capturingProvider) Type() string { return "capture" }

func (p *capturingProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return p.complete(ctx, req)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
