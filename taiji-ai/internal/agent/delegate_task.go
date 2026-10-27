package agent

import (
	"context"
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

const delegateTaskToolName = "delegate_task"

var defaultDelegateToolsAllow = []string{
	"list_dir", "read_file", "file_outline", "glob_search", "workspace_find", "grep", "rag_search", "web_fetch", "git_inspect",
}

const defaultSubAgentSystemPrompt = `You are a focused research sub-agent for Taiji.

Use the available read-only tools to gather facts from the workspace or indexed docs.
When you have enough information, reply with a concise summary for the parent agent.
Do not ask the user questions. Do not attempt file writes or shell commands.`

// DelegateTaskConfig wires delegate_task into the tool registry.
type DelegateTaskConfig struct {
	Providers *provider.Registry
	MainTools *tools.Registry
	Config    *taijicfg.Config
	WorkDir   string
}

// EnableDelegateTask registers delegate_task when dependencies are present.
func EnableDelegateTask(reg *tools.Registry, cfg DelegateTaskConfig) {
	if reg == nil || cfg.Providers == nil || cfg.MainTools == nil {
		return
	}
	reg.Register(DelegateTask{cfg: cfg})
}

// DelegateTask runs an isolated short agent loop for sub-tasks.
type DelegateTask struct {
	cfg DelegateTaskConfig
}

func (DelegateTask) Name() string { return delegateTaskToolName }

func (DelegateTask) Description() string {
	return "Run an isolated read-only sub-agent (max 8 steps) for research or exploration. Returns only the final summary to keep the main chat context small."
}

func (DelegateTask) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "Task for the sub-agent",
			},
			"tools_allow": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional tool names (default: list_dir, read_file, glob_search, grep, rag_search)",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t DelegateTask) Run(ctx context.Context, args map[string]any) (string, error) {
	prompt, _ := args["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	allow := parseToolsAllow(args)
	subReg := tools.FilterRegistry(t.cfg.MainTools, allow)

	parentSession := builtin.ToolSessionIDFrom(ctx)
	release := delegateSlots.acquire(parentSession)
	defer release()
	subCtx, _ := subSessionContext(ctx, parentSession)

	res, err := Run(subCtx, RunRequest{
		Goal:      prompt,
		Providers: t.cfg.Providers,
		Tools:     subReg,
		Config:    t.cfg.Config,
		WorkDir:   t.cfg.WorkDir,
		SubRun: SubRunConfig{
			Enabled:          true,
			MaxSteps:         8,
			SkipTurnRouter:   true,
			SkipHooks:        true,
			SkipMemoryPasses: true,
			SystemPrompt:     defaultSubAgentSystemPrompt,
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Answer), nil
}

func parseToolsAllow(args map[string]any) []string {
	raw, ok := args["tools_allow"]
	if !ok {
		return defaultDelegateToolsAllow
	}
	switch v := raw.(type) {
	case []string:
		if len(v) == 0 {
			return defaultDelegateToolsAllow
		}
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		if len(out) == 0 {
			return defaultDelegateToolsAllow
		}
		return out
	default:
		return defaultDelegateToolsAllow
	}
}

