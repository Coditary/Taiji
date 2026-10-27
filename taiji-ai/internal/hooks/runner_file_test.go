package hooks

import (
	"context"
	"testing"
)

func TestRunFileChangedInvokesPlugin(t *testing.T) {
	var files []string
	RegisterPlugin("file-hook", func(ctx context.Context, input ShellInput) (Result, error) {
		if input.File != nil {
			files = append(files, input.File.Files...)
		}
		return Result{}, nil
	})
	runner := NewRunner([]Entry{{
		ID: "fc", Event: EventFileChanged, Actions: []ActionSpec{{Plugin: "file-hook"}},
	}})
	err := runner.RunFileChanged(context.Background(), FileChangedData{
		Env: Env{WorkDir: t.TempDir()}, Tool: "write_file", Files: []string{"a.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "a.go" {
		t.Fatalf("files=%v", files)
	}
}
