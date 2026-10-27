package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFilesMergeAndDisable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.yaml")
	if err := os.WriteFile(path, []byte(`
version: 1
hooks:
  - id: one
    disable: true
    event: prompt.before
    actions:
      - plugin: prompt-hook
  - id: two
    event: context.before
    actions:
      - plugin: ctx-hook
`), 0o644); err != nil {
		t.Fatal(err)
	}

	RegisterPlugin("prompt-hook", func(ctx context.Context, input ShellInput) (Result, error) {
		return Result{UpdatedPrompt: "from-one"}, nil
	})
	RegisterPlugin("ctx-hook", func(ctx context.Context, input ShellInput) (Result, error) {
		return Result{AdditionalSystem: "from-two"}, nil
	})

	runner, err := LoadFiles([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if !runner.Enabled() {
		t.Fatal("expected runner")
	}
	prompt, err := runner.RunPromptBefore(context.Background(), PromptData{Env: Env{WorkDir: dir}, Text: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if prompt.Text != "x" {
		t.Fatalf("disabled hook ran: prompt=%q", prompt.Text)
	}
	ctxOut, err := runner.RunContextBefore(context.Background(), ContextData{Env: Env{WorkDir: dir}})
	if err != nil {
		t.Fatal(err)
	}
	if ctxOut.AdditionalSystem != "from-two" {
		t.Fatalf("system=%q", ctxOut.AdditionalSystem)
	}
}

func TestDefaultSearchPathsIncludesProjectAndGlobal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	paths := DefaultSearchPaths("/proj")
	if len(paths) < 2 {
		t.Fatalf("paths=%v", paths)
	}
	if paths[0] != filepath.Join("/proj", ".taiji", "hooks.yaml") {
		t.Fatalf("first=%q", paths[0])
	}
}
