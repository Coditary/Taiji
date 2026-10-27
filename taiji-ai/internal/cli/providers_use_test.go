package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestProvidersUseCommand(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root: root,
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
		},
	}
	app := &App{Config: cfg}
	cmd := newProvidersCmd(app)
	useCmd, _, err := cmd.Find([]string{"use"})
	if err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	runErr := useCmd.RunE(useCmd, []string{"dummy_a"})
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	if cfg.DefaultProvider != "dummy_a" || !strings.Contains(string(out), "dummy_a") {
		t.Fatalf("provider=%q out=%q", cfg.DefaultProvider, out)
	}
}
