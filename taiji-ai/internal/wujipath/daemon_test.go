package wujipath

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveCoreBinaryFromModuleTree(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "wuji-core")
	binDir := filepath.Join(core, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(core, "go.mod"), []byte(coreModule+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(binDir, "wuji-core")
	if err := os.WriteFile(bin, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WUJI_CORE_BIN", bin)
	got, err := ResolveCoreBinary("")
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Fatalf("got %q want %q", got, bin)
	}
}

func TestResolveCoreBinaryFromDevTreeWithMinimalPATH(t *testing.T) {
	home := t.TempDir()
	devRoot := filepath.Join(home, "Dev", "Coditary", "core", "wuji-core", "bin")
	if err := os.MkdirAll(devRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(devRoot, "wuji-core")
	if err := os.WriteFile(bin, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("WUJI_CORE_BIN", "")

	got, err := ResolveCoreBinary(home)
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Fatalf("got %q want %q", got, bin)
	}
}

func TestEnsureCoreDaemonIntegration(t *testing.T) {
	if os.Getenv("WUJI_INTEGRATION") == "" {
		t.Skip("set WUJI_INTEGRATION=1")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(home, ".wuji", "run", "drivers", "core.sock")
	endpoint := "unix://" + sock
	_ = os.Remove(sock)
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("WUJI_CORE_BIN", "")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := EnsureCoreDaemon(ctx, home, endpoint); err != nil {
		t.Fatal(err)
	}
}
