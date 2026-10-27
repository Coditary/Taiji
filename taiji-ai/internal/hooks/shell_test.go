package hooks

import (
	"context"
	"strings"
	"testing"
)

func TestRunShellActionCommandJSON(t *testing.T) {
	spec := ActionSpec{Command: `python3 -c 'import json,sys; print(json.dumps({"updated_prompt":"hi"}))'`}
	out, err := runShellAction(context.Background(), t.TempDir(), spec, ShellInput{
		Event: EventPromptBefore,
		Env:   Env{WorkDir: t.TempDir()},
		Prompt: &PromptData{Text: "orig"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.UpdatedPrompt != "hi" {
		t.Fatalf("prompt=%q", out.UpdatedPrompt)
	}
}

func TestExpandTemplateSubstitutions(t *testing.T) {
	in := ShellInput{
		Env:  Env{WorkDir: "/w", SessionID: "s1"},
		Tool: &ToolData{Tool: "write_file"},
		File: &FileChangedData{Files: []string{"a.go", "b.go"}},
	}
	got := expandTemplate("{{work_dir}} {{session_id}} {{tool}} {{files}}", in)
	for _, part := range []string{"/w", "s1", "write_file", "a.go b.go"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
}
