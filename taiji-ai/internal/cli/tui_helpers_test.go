package cli

import (
	"testing"
)

func TestProjectShortcutPositionalArgs(t *testing.T) {
	if got := projectShortcutPositionalArgs(nil); got != nil {
		t.Fatalf("nil argv: %v", got)
	}
	if got := projectShortcutPositionalArgs([]string{"chat", "run"}); got != nil {
		t.Fatalf("subcommand: %v", got)
	}
	if got := projectShortcutPositionalArgs([]string{"--local", "./proj", "-v"}); len(got) != 1 || got[0] != "./proj" {
		t.Fatalf("flags: %v", got)
	}
	if got := projectShortcutPositionalArgs([]string{"myapp"}); len(got) != 1 || got[0] != "myapp" {
		t.Fatalf("positional: %v", got)
	}
}
