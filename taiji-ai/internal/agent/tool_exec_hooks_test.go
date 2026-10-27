package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestRunToolCallHookBeforeError(t *testing.T) {
	hooks.RegisterPlugin("agentToolBeforeErr", func(ctx context.Context, input hooks.ShellInput) (hooks.Result, error) {
		return hooks.Result{}, errors.New("deny tool")
	})
	runner := hooks.NewRunner([]hooks.Entry{{
		ID: "tb", Event: hooks.EventToolBefore, Actions: []hooks.ActionSpec{{Plugin: "agentToolBeforeErr"}},
	}})
	reg := tools.NewRegistry()
	reg.Register(builtin.ListDir{})
	out := runToolCall(context.Background(), runner, reg, hooks.Env{WorkDir: t.TempDir()}, provider.ToolCall{
		ID: "1", Name: "list_dir", Args: `{}`,
	})
	if out.err == nil || !strings.Contains(out.err.Error(), "deny tool") {
		t.Fatalf("outcome=%+v", out)
	}
}

func TestRunToolCallHookAfterUpdatesOutput(t *testing.T) {
	hooks.RegisterPlugin("agentToolAfterSuffix", func(ctx context.Context, input hooks.ShellInput) (hooks.Result, error) {
		return hooks.Result{UpdatedOutput: input.Tool.Output + "++"}, nil
	})
	runner := hooks.NewRunner([]hooks.Entry{{
		ID: "ta", Event: hooks.EventToolAfter, Actions: []hooks.ActionSpec{{Plugin: "agentToolAfterSuffix"}},
	}})
	reg := tools.NewRegistry()
	reg.Register(builtin.ListDir{})
	out := runToolCall(context.Background(), runner, reg, hooks.Env{WorkDir: t.TempDir()}, provider.ToolCall{
		ID: "1", Name: "list_dir", Args: `{}`,
	})
	if out.err != nil || !strings.HasSuffix(out.result.Text, "++") {
		t.Fatalf("outcome=%+v", out)
	}
}

func TestRunToolCallDisabledBuiltin(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(builtin.ReadFile{})
	rt := session.AgentRuntime{BuiltinToolsAllow: []string{"list_dir"}}
	ctx := session.WithRunContext(context.Background(), rt)
	out := runToolCall(ctx, nil, reg, hooks.Env{}, provider.ToolCall{
		ID: "1", Name: "read_file", Args: `{"path":"x"}`,
	})
	if out.err == nil || !strings.Contains(out.err.Error(), "disabled") {
		t.Fatalf("outcome=%+v", out)
	}
}

func TestRunToolCallDisabledMCPTool(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(fakeMCPTool{fakeTool: fakeTool{name: "search"}, server: "ctx7"})
	rt := session.AgentRuntime{
		MCPServers:   []string{"ctx7"},
		MCPToolsDeny: []string{"ctx7:search"},
	}
	ctx := session.WithRunContext(context.Background(), rt)
	out := runToolCall(ctx, nil, reg, hooks.Env{}, provider.ToolCall{
		ID: "1", Name: "ctx7__search", Args: `{}`,
	})
	if out.err == nil || !strings.Contains(out.err.Error(), "disabled") {
		t.Fatalf("outcome=%+v", out)
	}
}

func TestExecuteToolCallsUnknownTool(t *testing.T) {
	out := executeToolCalls(context.Background(), nil, tools.NewRegistry(), hooks.Env{}, []provider.ToolCall{
		{ID: "1", Name: "missing_tool", Args: `{}`},
	}, nil)
	if len(out) != 1 || out[0].err == nil {
		t.Fatalf("outcome=%+v", out[0])
	}
}
