package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromPathsJSONAndYAML(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "claude.json")
	yamlPath := filepath.Join(dir, "hooks.yaml")
	if err := os.WriteFile(jsonPath, []byte(`{
  "version": 1,
  "hooks": {
    "Stop": [{
      "hooks": [{ "type": "command", "command": "echo stop" }]
    }]
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	RegisterPlugin("yaml-hook", func(ctx context.Context, input ShellInput) (Result, error) {
		return Result{AdditionalSystem: "yaml"}, nil
	})
	if err := os.WriteFile(yamlPath, []byte(`
version: 1
hooks:
  - id: yaml-one
    event: context.before
    actions:
      - plugin: yaml-hook
`), 0o644); err != nil {
		t.Fatal(err)
	}
	runner, err := LoadFromPaths([]string{jsonPath, yamlPath})
	if err != nil {
		t.Fatal(err)
	}
	if runner == nil || !runner.Enabled() {
		t.Fatal("expected merged runner")
	}
	out, err := runner.RunContextBefore(context.Background(), ContextData{Env: Env{WorkDir: dir}})
	if err != nil {
		t.Fatal(err)
	}
	if out.AdditionalSystem != "yaml" {
		t.Fatalf("system=%q", out.AdditionalSystem)
	}
}
