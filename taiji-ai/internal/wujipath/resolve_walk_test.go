package wujipath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindModuleDirWalksUpToCore(t *testing.T) {
	core := filepath.Join(t.TempDir(), "wuji-core")
	nested := filepath.Join(core, "pkg", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(core, "go.mod"), []byte(coreModule+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, ok := findModuleDir(nested)
	if !ok || dir != core {
		t.Fatalf("findModuleDir=%q ok=%v want %q", dir, ok, core)
	}
}

func TestModuleDirBesideBinaryAndSiblingCore(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	bin := filepath.Join(app, "bin", "wuji")
	core := filepath.Join(root, "wuji-core")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(core, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(core, "go.mod"), []byte(coreModule+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, ok := moduleDirBesideBinary(bin)
	if !ok || dir != core {
		t.Fatalf("beside binary: dir=%q ok=%v", dir, ok)
	}
}

func TestResolveRootAbsoluteAndHomeFallback(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "wuji-runtime")
	got := ResolveRoot("/cfg", abs)
	if got != filepath.Clean(abs) {
		t.Fatalf("abs=%q", got)
	}
	t.Setenv("WUJI_ROOT", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	got = ResolveRoot("", "")
	if got != home {
		t.Fatalf("home fallback=%q", got)
	}
}

func TestIsWujiAIModuleWithDotWuji(t *testing.T) {
	root := t.TempDir()
	wujiAI := filepath.Join(root, "wuji-ai")
	if err := os.MkdirAll(filepath.Join(wujiAI, ".wuji"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wujiAI, "go.mod"), []byte("module github.com/coditary/wuji-ai\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, ok := findModuleDir(wujiAI)
	if !ok || dir != filepath.Clean(wujiAI) {
		t.Fatalf("dir=%q ok=%v", dir, ok)
	}
}
