package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestThemeListCommand(t *testing.T) {
	root := newThemeCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "dark") {
		t.Fatalf("expected built-in themes in output: %q", text)
	}
}
