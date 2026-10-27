package hooks

import (
	"context"
	"testing"
)

func TestRunContextBeforeAppendsSystem(t *testing.T) {
	RegisterPlugin("ctx", func(ctx context.Context, input ShellInput) (Result, error) {
		return Result{AdditionalSystem: "extra rules"}, nil
	})
	runner := NewRunner([]Entry{{
		ID: "ctx", Event: EventContextBefore, Actions: []ActionSpec{{Plugin: "ctx"}},
	}})
	out, err := runner.RunContextBefore(context.Background(), ContextData{Env: Env{WorkDir: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	if out.AdditionalSystem != "extra rules" {
		t.Fatalf("system=%q", out.AdditionalSystem)
	}
}

func TestRunToolAfterUpdatesOutput(t *testing.T) {
	RegisterPlugin("suffix", func(ctx context.Context, input ShellInput) (Result, error) {
		return Result{UpdatedOutput: input.Tool.Output + " [redacted]"}, nil
	})
	runner := NewRunner([]Entry{{
		ID: "suffix", Event: EventToolAfter, Actions: []ActionSpec{{Plugin: "suffix"}},
	}})
	out, err := runner.RunToolAfter(context.Background(), ToolData{
		Env: Env{WorkDir: t.TempDir()}, Tool: "read_file", Output: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Output != "secret [redacted]" {
		t.Fatalf("output=%q", out.Output)
	}
}

func TestRunSessionLifecycle(t *testing.T) {
	var seen string
	RegisterPlugin("sess", func(ctx context.Context, input ShellInput) (Result, error) {
		if input.Session != nil {
			seen = input.Session.Answer
		}
		return Result{}, nil
	})
	runner := NewRunner([]Entry{{
		ID: "sess", Event: EventSessionIdle, Actions: []ActionSpec{{Plugin: "sess"}},
	}})
	if err := runner.RunSession(context.Background(), EventSessionIdle, SessionData{
		Env: Env{WorkDir: t.TempDir()}, Answer: "done",
	}); err != nil {
		t.Fatal(err)
	}
	if seen != "done" {
		t.Fatalf("seen=%q", seen)
	}
}

func TestDecodeShellResultPlainText(t *testing.T) {
	res, err := decodeShellResult("  logged\n")
	if err != nil {
		t.Fatal(err)
	}
	if res.AdditionalContext != "logged" {
		t.Fatalf("ctx=%q", res.AdditionalContext)
	}
}
