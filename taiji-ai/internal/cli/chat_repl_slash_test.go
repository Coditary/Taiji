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

func TestRunChatREPLSlashCompactAndTools(t *testing.T) {
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

	oldErr := os.Stderr
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = errW
	defer func() { os.Stderr = oldErr }()

	var errBuf bytes.Buffer
	go func() {
		_, _ = io.Copy(&errBuf, errR)
	}()

	script := "/tools\n/help\n/exit\n"
	go func() {
		_, _ = io.WriteString(inW, script)
		_ = inW.Close()
	}()

	if err := runChatREPL(context.Background(), app, chatOptions{
		runtimeOpts: runtime.Options{Quiet: true, Local: true, LocalOnly: true},
	}); err != nil {
		t.Fatal(err)
	}
	_ = errW.Close()
	text := errBuf.String()
	if !strings.Contains(text, "list_dir") && !strings.Contains(text, "(no tools") {
		t.Fatalf("tools output missing: %q", text)
	}
}
