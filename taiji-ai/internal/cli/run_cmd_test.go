package cli

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
)

func TestRunCommandDummyProvider(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy_a",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
		},
	}
	app := &App{Config: cfg}
	cmd := newRunCmd(app)
	cmd.SetContext(context.Background())
	oldErr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	oldOut := os.Stdout
	outR, outW, _ := os.Pipe()
	os.Stdout = outW
	runErr := cmd.RunE(cmd, []string{"Ping"})
	_ = w.Close()
	_ = outW.Close()
	os.Stderr = oldErr
	os.Stdout = oldOut
	errText, _ := io.ReadAll(r)
	answer, _ := io.ReadAll(outR)
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !strings.Contains(string(answer), "Pong") {
		t.Fatalf("answer=%q", answer)
	}
	if !strings.Contains(string(errText), "session:") {
		t.Fatalf("stderr=%q", errText)
	}
}

func TestRunCommandResumesSession(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy_a",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
		},
	}
	sess, err := session.New(root, "fixed-id", "dummy_a", "prior")
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}
	app := &App{Config: cfg}
	cmd := newRunCmd(app)
	_ = cmd.Flags().Set("session", "fixed-id")
	cmd.SetContext(context.Background())
	oldOut := os.Stdout
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	os.Stdout = devNull
	defer func() { os.Stdout = oldOut; _ = devNull.Close() }()
	if err := cmd.RunE(cmd, []string{"Ping"}); err != nil {
		t.Fatal(err)
	}
	_ = runtime.Options{Quiet: true, Local: true, LocalOnly: true}
}
