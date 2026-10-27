package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/runtime"
)

func TestProviderToolChoiceDirect(t *testing.T) {
	if os.Getenv("TAIJI_TOOL_BENCH") == "" {
		t.Skip("set TAIJI_TOOL_BENCH=1 to run live tool-trigger benchmark")
	}
	root, err := findTaijiRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	cfg, err := taijicfg.Load(taijicfg.LoadOptions{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rt, err := runtime.New(ctx, cfg, runtime.Options{Local: true, LocalOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	p, err := rt.Providers.Get("gemma4")
	if err != nil {
		t.Fatal(err)
	}
	defs := make([]provider.ToolDef, 0, len(rt.Tools.Definitions()))
	for _, d := range rt.Tools.Definitions() {
		defs = append(defs, provider.ToolDef{Name: d.Name, Namespace: d.Namespace, Description: d.Description, Parameters: d.Parameters})
	}
	resp, err := p.Complete(ctx, provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: taijicfg.DefaultSystemPrompt()},
			{Role: provider.RoleUser, Content: "List files in the current directory"},
		},
		Tools:         defs,
		ContextWindow: 32768,
		ToolChoice:    "required",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) == 0 {
		t.Fatalf("direct provider call: expected tool_calls, got content=%q finish=%q usage=%+v", resp.Content, resp.FinishReason, resp.Usage)
	}
	t.Logf("direct tool call: %s %s", resp.ToolCalls[0].Name, resp.ToolCalls[0].Args)
}

func TestToolTriggerBench(t *testing.T) {
	if os.Getenv("TAIJI_TOOL_BENCH") == "" {
		t.Skip("set TAIJI_TOOL_BENCH=1 to run live tool-trigger benchmark")
	}

	root, err := findTaijiRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	cfg, err := taijicfg.Load(taijicfg.LoadOptions{Local: true})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	rt, err := runtime.New(ctx, cfg, runtime.Options{Local: true, LocalOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	goals := []string{
		"List files in the current directory",
		"Show me the project structure",
		"Find all Go files in this project",
		"Grep for func Register in the codebase",
		"Read the Makefile",
		"List files in internal/tools",
		"Search for tool_call in Go files",
		"List the contents of cmd/",
		"Find config.yaml files",
		"Show files in internal/agent",
	}

	triggered := 0
	for i, goal := range goals {
		var sawTool bool
		var err error
		for attempt := 1; attempt <= 2; attempt++ {
			sawTool = false
			_, err = Run(ctx, RunRequest{
				Goal:       goal,
				ProviderID: "gemma4",
				Providers:  rt.Providers,
				Tools:      rt.Tools,
				Config:     cfg,
				WorkDir:    root,
				EventHandler: func(ev events.Event) {
					if ev.Type == events.TypeToolCall {
						sawTool = true
					}
				},
			})
			if err == nil {
				break
			}
			if attempt == 2 {
				t.Fatalf("run %d (%q): %v", i+1, goal, err)
			}
			t.Logf("run %d attempt %d retry after: %v", i+1, attempt, err)
		}
		if sawTool {
			triggered++
		}
		t.Logf("run %d: tool_triggered=%v goal=%q", i+1, sawTool, goal)
	}

	rate := float64(triggered) / float64(len(goals)) * 100
	t.Logf("tool trigger rate: %d/%d (%.0f%%)", triggered, len(goals), rate)
	if rate < 80 {
		t.Fatalf("tool trigger rate %.0f%% below 80%% target", rate)
	}
}

func findTaijiRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, ".taiji")); err == nil {
				return dir, nil
			}
		}
		if dir == filepath.Dir(dir) {
			break
		}
	}
	return wd, nil
}
