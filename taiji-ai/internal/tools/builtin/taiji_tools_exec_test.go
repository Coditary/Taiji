package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunTaijiToolsViaFakeBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "taiji-tools")
	script := "#!/bin/sh\nif [ \"$1\" = \"glob\" ]; then echo \"found.go\"; exit 0; fi\necho \"ok-$1\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TAIJI_TOOLS", bin)
	t.Setenv("TAIJI_GLOBWALK", "")

	out, err := runTaijiTools(context.Background(), "echo-test")
	if err != nil || out != "ok-echo-test\n" {
		t.Fatalf("run: err=%v out=%q", err, out)
	}
	stdinOut, err := runTaijiToolsStdin(context.Background(), "<html>x</html>", "html-to-text")
	if err != nil || !strings.Contains(stdinOut, "ok-html-to-text") {
		t.Fatalf("stdin: err=%v out=%q", err, stdinOut)
	}
	lines, err := runGlobwalkBinary(context.Background(), dir, "*.go", 10)
	if err != nil || len(lines) != 1 || lines[0] != "found.go" {
		t.Fatalf("glob: err=%v lines=%v", err, lines)
	}
}

func TestRunTaijiToolsMissingBinary(t *testing.T) {
	t.Setenv("TAIJI_TOOLS", "/nonexistent/taiji-tools")
	t.Setenv("TAIJI_GLOBWALK", "")
	if _, err := runTaijiTools(context.Background()); err == nil {
		t.Fatal("expected missing binary error")
	}
}
