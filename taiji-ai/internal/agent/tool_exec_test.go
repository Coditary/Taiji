package agent

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestExecuteToolCallsRunsBuiltin(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(builtin.ListDir{})
	env := hooks.Env{WorkDir: t.TempDir(), Goal: "test", Step: 1}
	out := executeToolCalls(context.Background(), nil, reg, env, []provider.ToolCall{
		{ID: "1", Name: "list_dir", Args: `{}`},
	}, nil)
	if len(out) != 1 || out[0].err != nil || out[0].result.Text == "" {
		t.Fatalf("outcome=%+v", out[0])
	}
}

func TestExecuteToolCallsDetectsDuplicate(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(builtin.ListDir{})
	msgs := []provider.Message{
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "1", Name: "list_dir", Content: "ok"},
	}
	out := executeToolCalls(context.Background(), nil, reg, hooks.Env{}, []provider.ToolCall{
		{ID: "2", Name: "list_dir", Args: `{}`},
	}, msgs)
	if len(out) != 1 || !out[0].duplicate {
		t.Fatalf("outcome=%+v", out[0])
	}
}
