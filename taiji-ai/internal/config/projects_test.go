package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVisibleProjectsHidesLocalFromOtherRoots(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	parent := filepath.Join(t.TempDir(), "Taiji")
	local := filepath.Join(parent, "nested")
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

	if err := RegisterProject(parent, false); err != nil {
		t.Fatal(err)
	}
	if err := RegisterProject(local, true); err != nil {
		t.Fatal(err)
	}

	fromParent, err := VisibleProjects(parent, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(fromParent) != 1 || cleanProjectRoot(fromParent[0].Root) != cleanProjectRoot(parent) {
		t.Fatalf("parent view = %#v, want only parent", fromParent)
	}

	fromLocal, err := VisibleProjects(local, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(fromLocal) != 1 || cleanProjectRoot(fromLocal[0].Root) != cleanProjectRoot(local) {
		t.Fatalf("local view = %#v, want only local", fromLocal)
	}
}
