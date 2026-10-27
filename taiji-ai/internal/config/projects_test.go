package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterProjectPreservesLocalFolderFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	home := filepath.Join(t.TempDir(), "Leodoras")
	if err := RegisterProject(home, true); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(SessionsDir(home), "s1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(SessionsDir(home), "s1", "state.json"), []byte(`{"id":"s1","messages":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Opening a session in a folder workspace must not clear the folder flag (embed used to pass localMode=false).
	if err := RegisterProject(home, false); err != nil {
		t.Fatal(err)
	}

	folders, err := VisibleFolders(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || cleanProjectRoot(folders[0].Root) != cleanProjectRoot(home) {
		t.Fatalf("folders = %#v, want home %q", folders, home)
	}
}

func TestVisibleFoldersIncludesRegisteredFolderWithoutSessions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	home := filepath.Join(t.TempDir(), "Leodoras")
	if err := RegisterProject(home, true); err != nil {
		t.Fatal(err)
	}

	folders, err := VisibleFolders("")
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || cleanProjectRoot(folders[0].Root) != cleanProjectRoot(home) {
		t.Fatalf("folders = %#v, want registered empty folder %q", folders, home)
	}
}

func TestVisibleProjectsHidesLocalFromOtherRoots(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	parent := filepath.Join(t.TempDir(), "Taiji")
	local := filepath.Join(parent, "nested")
	if err := RegisterProject(parent, false); err != nil {
		t.Fatal(err)
	}
	if err := RegisterProject(local, true); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(SessionsDir(parent), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(SessionsDir(local), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(SessionsDir(parent), "p1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(SessionsDir(parent), "p1", "state.json"), []byte(`{"id":"p1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(SessionsDir(local), "l1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(SessionsDir(local), "l1", "state.json"), []byte(`{"id":"l1"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	fromParent, err := VisibleProjects(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(fromParent) != 1 || cleanProjectRoot(fromParent[0].Root) != cleanProjectRoot(parent) {
		t.Fatalf("parent view = %#v, want only parent", fromParent)
	}

	fromLocal, err := VisibleFolders(local)
	if err != nil {
		t.Fatal(err)
	}
	if len(fromLocal) != 1 || cleanProjectRoot(fromLocal[0].Root) != cleanProjectRoot(local) {
		t.Fatalf("local view = %#v, want only local", fromLocal)
	}
}
