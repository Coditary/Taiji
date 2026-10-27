package agent

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestProjectWorkspaceRootPrefersWorkDir(t *testing.T) {
	cfg := &taijicfg.Config{Root: "/data/project"}
	req := RunRequest{
		Config:  cfg,
		WorkDir: "/data/project/src",
	}
	if got := projectWorkspaceRoot(req); got != "/data/project/src" {
		t.Fatalf("projectWorkspaceRoot = %q, want work dir", got)
	}
}

func TestProjectWorkspaceRootFallsBackToConfigRoot(t *testing.T) {
	cfg := &taijicfg.Config{Root: "/data/project"}
	req := RunRequest{Config: cfg}
	if got := projectWorkspaceRoot(req); got != "/data/project" {
		t.Fatalf("projectWorkspaceRoot = %q, want config root", got)
	}
}
