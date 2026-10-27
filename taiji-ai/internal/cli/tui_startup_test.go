package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConsumeTUIProjectArgDot(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	rest, project, newSession := consumeTUIProjectArg([]string{"."})
	if rest != nil {
		t.Fatalf("rest = %v, want nil", rest)
	}
	if !newSession {
		t.Fatal("expected new session")
	}
	want, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	if project != want {
		t.Fatalf("project = %q, want %q", project, want)
	}
}

func TestConsumeTUIProjectArgIgnoresFlags(t *testing.T) {
	args := []string{"--help"}
	rest, project, newSession := consumeTUIProjectArg(args)
	if rest == nil || len(rest) != 1 || project != "" || newSession {
		t.Fatalf("got rest=%v project=%q new=%v", rest, project, newSession)
	}
}

func TestProjectShortcutPositionalArgsSkipsSubcommands(t *testing.T) {
	if got := projectShortcutPositionalArgs([]string{"tui", "."}); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
	if got := projectShortcutPositionalArgs([]string{"--local", "."}); len(got) != 1 || got[0] != "." {
		t.Fatalf("got %v, want [.]", got)
	}
}

func TestConsumeTUIProjectArgNestedDir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "pkg")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	rest, project, newSession := consumeTUIProjectArg([]string{sub})
	if rest != nil || !newSession {
		t.Fatalf("rest=%v new=%v", rest, newSession)
	}
	want, err := filepath.Abs(sub)
	if err != nil {
		t.Fatal(err)
	}
	if project != want {
		t.Fatalf("project = %q, want %q", project, want)
	}
}
