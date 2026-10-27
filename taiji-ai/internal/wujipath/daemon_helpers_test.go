package wujipath

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUnixSocketPathAndReady(t *testing.T) {
	if unixSocketPath("tcp://x") != "" {
		t.Fatal("non-unix endpoint")
	}
	sock := filepath.Join(t.TempDir(), "core.sock")
	if socketReady(sock) {
		t.Fatal("missing socket should not be ready")
	}
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if !socketReady(sock) {
		t.Fatal("expected ready listener")
	}
}

func TestExecutableFileChecks(t *testing.T) {
	if ok, _ := executableFile(""); ok {
		t.Fatal("empty path")
	}
	dir := t.TempDir()
	if ok, _ := executableFile(dir); ok {
		t.Fatal("directory")
	}
	bin := filepath.Join(dir, "tool")
	if err := os.WriteFile(bin, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, _ := executableFile(bin); ok {
		t.Fatal("not executable bit")
	}
	if err := os.Chmod(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if ok, err := executableFile(bin); !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestEnsureCoreDaemonWithFakeCore(t *testing.T) {
	root := t.TempDir()
	sock := filepath.Join(root, "run", "core.sock")
	endpoint := "unix://" + sock
	if err := os.MkdirAll(filepath.Dir(sock), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(binDir, "wuji-core")
	body := `#!/bin/sh
set -eu
addr="$2"
sock="${addr#unix://}"
exec python3 - "$sock" <<'PY'
import socket, sys, time, os
sock = sys.argv[1]
os.makedirs(os.path.dirname(sock), exist_ok=True)
if os.path.exists(sock):
    os.unlink(sock)
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.bind(sock)
s.listen(8)
time.sleep(120)
PY
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WUJI_CORE_BIN", script)
	t.Setenv("HOME", root)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := EnsureCoreDaemon(ctx, root, endpoint); err != nil {
		t.Fatal(err)
	}
	if !socketReady(sock) {
		t.Fatal("daemon did not expose socket")
	}
}
