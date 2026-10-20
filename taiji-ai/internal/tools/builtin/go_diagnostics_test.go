package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoSyntaxDiagnosticsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.go")
	if err := os.WriteFile(path, []byte("package p\nfunc {\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := goSyntaxDiagnostics(path)
	if msg == "" || !strings.Contains(msg, "go/parser") {
		t.Fatalf("expected parser error, got %q", msg)
	}
}

func TestEnrichResultSkipsNonGo(t *testing.T) {
	out := enrichResultWithGoDiagnostics("readme.md", "ok")
	if out != "ok" {
		t.Fatalf("unexpected: %q", out)
	}
}
