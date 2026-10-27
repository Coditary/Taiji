package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/runtime"
)

func TestRunChatREPLDummyOneTurnAndExit(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy_a",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
		},
	}
	app := &App{Config: cfg}

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldIn := os.Stdin
	os.Stdin = inR
	defer func() { os.Stdin = oldIn; _ = inR.Close() }()

	var stdout bytes.Buffer
	oldOut := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = wOut
	defer func() { os.Stdout = oldOut; _ = wOut.Close() }()

	go func() {
		_, _ = io.WriteString(inW, "Ping\n/exit\n")
		_ = inW.Close()
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runChatREPL(context.Background(), app, chatOptions{
			runtimeOpts: runtime.Options{Quiet: true, Local: true, LocalOnly: true},
		})
	}()

	// Drain stdout while REPL runs.
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdout, rOut)
		close(done)
	}()

	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	_ = wOut.Close()
	<-done

	out := stdout.String()
	if !strings.Contains(out, "Pong") && !strings.Contains(out, ">>>") {
		t.Fatalf("stdout=%q", out)
	}
}
