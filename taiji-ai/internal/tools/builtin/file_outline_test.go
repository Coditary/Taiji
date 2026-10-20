package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutlineGoSource(t *testing.T) {
	src := `package demo

type Widget struct {
	Name string
}

func Hello() {
	println("hi")
}

func (w *Widget) Run() error {
	return nil
}
`
	out, err := outlineGoSource("demo.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "type Widget") {
		t.Fatalf("missing type: %q", out)
	}
	if !strings.Contains(out, "func Hello") {
		t.Fatalf("missing func: %q", out)
	}
	if !strings.Contains(out, "func (w *Widget) Run") {
		t.Fatalf("missing method: %q", out)
	}
	if !strings.Contains(out, "L") {
		t.Fatalf("expected line ranges: %q", out)
	}
}

func TestFileOutlineToolGo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.go")
	content := "package p\n\nfunc Foo() {}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (FileOutline{}).Run(t.Context(), map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "func Foo") {
		t.Fatalf("unexpected: %q", out)
	}
}
