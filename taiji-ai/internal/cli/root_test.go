package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFlagSetsConfigRootToCWD(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	taijiDir := filepath.Join(project, ".taiji")
	if err := os.MkdirAll(taijiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taijiDir, "config.yaml"), []byte("default_provider: parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(project, "sub", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	app, err := New()
	if err != nil {
		t.Fatal(err)
	}
	app.Root.SetArgs([]string{"--local", "providers", "list"})
	if err := app.Root.Execute(); err != nil {
		t.Fatal(err)
	}
	if app.Config == nil {
		t.Fatal("config not loaded")
	}
	wantRoot, err := filepath.Abs(nested)
	if err != nil {
		t.Fatal(err)
	}
	if app.Config.Root != wantRoot {
		t.Fatalf("root = %q, want %q", app.Config.Root, wantRoot)
	}
}
