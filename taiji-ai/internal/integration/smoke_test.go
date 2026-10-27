package integration_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/agent"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func dummyConfig(t *testing.T) *taijicfg.Config {
	root := t.TempDir()
	return &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy": {Type: "dummy", Model: "alpha"},
		},
		Wuji: taijicfg.WujiConfig{AutoSync: false},
		Skills: taijicfg.SkillsConfig{
			Dirs: []string{filepath.Join(root, ".taiji", "skills")},
		},
		MCP: taijicfg.MCPConfig{
			ConfigPath: filepath.Join(root, ".wuji", "mcp.json"),
		},
	}
}

func TestRuntimeAndAgentDummySmoke(t *testing.T) {
	cfg := dummyConfig(t)
	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Quiet: true, Local: true, LocalOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	sess, err := session.New(cfg.Root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run(context.Background(), agent.RunRequest{
		Goal:       "Ping",
		ProviderID: "dummy",
		Session:    sess,
		Providers:  rt.Providers,
		Tools:      rt.Tools,
		Config:     cfg,
		SubRun:     agent.SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer != "Pong" {
		t.Fatalf("answer=%q", result.Answer)
	}
	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestProviderRegistryDummyOnly(t *testing.T) {
	cfg := dummyConfig(t)
	reg, err := provider.NewRegistry(context.Background(), cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	p, err := reg.Get("dummy")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), provider.CompletionRequest{
		Messages: []provider.Message{{Role: provider.RoleUser, Content: "Hallo"}},
	})
	if err != nil || resp.Content != "Hallo" {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}

func TestAgentWithStaticTools(t *testing.T) {
	cfg := dummyConfig(t)
	reg, err := provider.NewRegistry(context.Background(), cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = agent.Run(context.Background(), agent.RunRequest{
		Goal:       "Hallo",
		ProviderID: "dummy",
		Providers:  reg,
		Tools:      tools.NewRegistry(),
		Config:     cfg,
		SubRun:     agent.SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		t.Fatal(err)
	}
}
