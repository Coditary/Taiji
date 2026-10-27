package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestRunToolCallFileChangedHook(t *testing.T) {
	var changed []string
	hooks.RegisterPlugin("agentFileChanged", func(ctx context.Context, input hooks.ShellInput) (hooks.Result, error) {
		if input.File != nil {
			changed = append(changed, input.File.Files...)
		}
		return hooks.Result{}, nil
	})
	runner := hooks.NewRunner([]hooks.Entry{{
		ID: "fc", Event: hooks.EventFileChanged, Actions: []hooks.ActionSpec{{Plugin: "agentFileChanged"}},
	}})

	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	args, _ := json.Marshal(map[string]string{"path": path, "content": "hi"})
	reg := tools.NewRegistry()
	reg.Register(builtin.WriteFile{})

	out := runToolCall(context.Background(), runner, reg, hooks.Env{WorkDir: dir}, provider.ToolCall{
		ID:   "w1",
		Name: "write_file",
		Args: string(args),
	})
	if out.err != nil {
		t.Fatal(out.err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if len(changed) == 0 {
		t.Fatal("expected file changed hook")
	}
}
