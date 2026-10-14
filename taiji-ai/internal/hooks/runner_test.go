package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunToolBeforeBlocks(t *testing.T) {
	RegisterPlugin("block-shell", func(ctx context.Context, input ShellInput) (Result, error) {
		if input.Tool != nil && input.Tool.Tool == "shell" {
			return Result{Block: true, Message: "shell disabled"}, nil
		}
		return Result{}, nil
	})

	runner := NewRunner([]Entry{{
		ID:    "block-shell",
		Event: EventToolBefore,
		Actions: []ActionSpec{{Plugin: "block-shell"}},
	}})

	_, err := runner.RunToolBefore(context.Background(), ToolData{
		Env:  Env{WorkDir: "/tmp"},
		Tool: "shell",
		Args: `{"command":"echo hi"}`,
	})
	if err == nil {
		t.Fatal("expected block error")
	}
}

func TestRunPromptBeforeUpdatesText(t *testing.T) {
	RegisterPlugin("prefix", func(ctx context.Context, input ShellInput) (Result, error) {
		if input.Prompt == nil {
			return Result{}, nil
		}
		return Result{UpdatedPrompt: input.Prompt.Text + " [hook]"}, nil
	})

	runner := NewRunner([]Entry{{
		ID:      "prefix",
		Event:   EventPromptBefore,
		Actions: []ActionSpec{{Plugin: "prefix"}},
	}})

	out, err := runner.RunPromptBefore(context.Background(), PromptData{
		Env:  Env{WorkDir: t.TempDir()},
		Text: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "hello [hook]" {
		t.Fatalf("prompt = %q", out.Text)
	}
}

func TestLoadClaudeJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	raw := `{
  "version": 1,
  "hooks": {
    "Stop": [{
      "hooks": [{ "type": "command", "command": "echo done" }]
    }]
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadClaudeJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Event != EventSessionIdle {
		t.Fatalf("entries = %#v", entries)
	}
}
