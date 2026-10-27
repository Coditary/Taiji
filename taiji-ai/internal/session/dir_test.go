package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/config"
)

func TestDirBindingSaveLoad(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "project", "src")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := SaveDirBinding(root, workDir, "session-123"); err != nil {
		t.Fatal(err)
	}

	binding, ok, err := LoadDirBinding(root, workDir)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected binding")
	}
	if binding.SessionID != "session-123" {
		t.Fatalf("session id = %q", binding.SessionID)
	}
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if binding.WorkDir != absWorkDir {
		t.Fatalf("work dir = %q, want %q", binding.WorkDir, absWorkDir)
	}
}

func TestDirBindingUsesSeparateKeysPerDirectory(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	for _, dir := range []string{dirA, dirB} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := SaveDirBinding(root, dirA, "session-a"); err != nil {
		t.Fatal(err)
	}
	if err := SaveDirBinding(root, dirB, "session-b"); err != nil {
		t.Fatal(err)
	}

	bindingA, ok, err := LoadDirBinding(root, dirA)
	if err != nil || !ok {
		t.Fatalf("dir A: ok=%v err=%v", ok, err)
	}
	bindingB, ok, err := LoadDirBinding(root, dirB)
	if err != nil || !ok {
		t.Fatalf("dir B: ok=%v err=%v", ok, err)
	}
	if bindingA.SessionID != "session-a" || bindingB.SessionID != "session-b" {
		t.Fatalf("bindings: a=%q b=%q", bindingA.SessionID, bindingB.SessionID)
	}
}

func TestDirBindingStoredUnderRuntimeRoot(t *testing.T) {
	projectRoot := t.TempDir()
	localRoot := filepath.Join(projectRoot, "nested")
	if err := os.MkdirAll(localRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := config.RegisterProject(localRoot, true); err != nil {
		t.Fatal(err)
	}

	if err := SaveDirBinding(localRoot, localRoot, "local-session"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(config.RuntimeDir(localRoot), "dir-sessions")
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("binding files = %d, want 1", len(entries))
	}
}
