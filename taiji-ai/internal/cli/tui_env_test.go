package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyTUIStartupEnv(t *testing.T) {
	t.Setenv(envTUIStartProject, "")
	t.Setenv(envTUIStartNewSession, "")
	applyTUIStartupEnv("/tmp/project", true)
	if os.Getenv(envTUIStartProject) != "/tmp/project" {
		t.Fatalf("project=%q", os.Getenv(envTUIStartProject))
	}
	if os.Getenv(envTUIStartNewSession) != "1" {
		t.Fatal("expected new session flag")
	}
	applyTUIStartupEnv("", false)
	if os.Getenv(envTUIStartProject) != "/tmp/project" {
		t.Fatal("empty project should not clear existing")
	}
}

func TestResolveTUIExecutableFromPATH(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "taiji-tui")
	if err := os.WriteFile(bin, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin")
	got, err := resolveTUIExecutable()
	if err != nil || got != bin {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestTryLaunchTUIProjectShortcutSkipsSubcommand(t *testing.T) {
	ok, err := tryLaunchTUIProjectShortcut([]string{"chat"})
	if ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
