package builtin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitInspectStatusAndLog(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	SetWorkspaceRootsForTest(root, "")
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "t@example.com")
	runGit(t, root, "config", "user.name", "tester")
	path := filepath.Join(root, "hello.txt")
	if err := os.WriteFile(path, []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "hello.txt")
	runGit(t, root, "commit", "-m", "initial")

	status, err := (GitInspect{}).Run(context.Background(), map[string]any{"action": "status"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, "branch: master") && !strings.Contains(status, "branch: main") {
		t.Fatalf("status: %s", status)
	}
	logOut, err := (GitInspect{}).Run(context.Background(), map[string]any{
		"action": "log",
		"limit":  5,
	})
	if err != nil || !strings.Contains(logOut, "initial") {
		t.Fatalf("log: %v %s", err, logOut)
	}
	blame, err := (GitInspect{}).Run(context.Background(), map[string]any{
		"action":     "blame",
		"path":       "hello.txt",
		"start_line": 1,
		"end_line":   1,
	})
	if err != nil || !strings.Contains(blame, "hi") {
		t.Fatalf("blame: %v %s", err, blame)
	}
	show, err := (GitInspect{}).Run(context.Background(), map[string]any{
		"action": "show",
		"commit": "HEAD",
	})
	if err != nil || !strings.Contains(show, "initial") {
		t.Fatalf("show: %v %s", err, show)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}
