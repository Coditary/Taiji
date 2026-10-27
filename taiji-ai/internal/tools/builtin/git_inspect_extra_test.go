package builtin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitInspectWithWorkspaceRoot(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	t.Cleanup(func() { SetWorkspaceRootsForTest(".", "") })
	SetWorkspaceRootsForTest(root, "")

	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "t@example.com")
	runGit(t, root, "config", "user.name", "tester")
	path := filepath.Join(root, "track.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "track.go")
	runGit(t, root, "commit", "-m", "add track")

	logOut, err := (GitInspect{}).Run(context.Background(), map[string]any{
		"action": "log",
		"path":   "track.go",
		"limit":  40,
	})
	if err != nil || !strings.Contains(logOut, "add track") {
		t.Fatalf("log: %v %s", err, logOut)
	}

	_, err = (GitInspect{}).Run(context.Background(), map[string]any{"action": "blame"})
	if err == nil || !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("blame validation: %v", err)
	}
}
