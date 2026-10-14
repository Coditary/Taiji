package wujipath

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coditary/wuji-core/pkg/netx"
)

// EnsureCoreDaemon starts wuji-core when the unix socket is missing.
func EnsureCoreDaemon(ctx context.Context, root, endpoint string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		root = home
	}
	if err := EnsureWujiConfig(root); err != nil {
		return err
	}

	sock := unixSocketPath(endpoint)
	if sock == "" {
		return nil
	}
	if socketReady(sock) {
		return nil
	}
	_ = os.Remove(sock)

	bin, err := ResolveCoreBinary(root)
	if err != nil {
		return err
	}

	logPath := filepath.Join(root, ".wuji", "logs", "core.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, bin, "--addr", endpoint)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), wujiRootEnv(root))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start wuji-core: %w", err)
	}
	_ = logFile.Close()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if socketReady(sock) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("wuji-core did not become ready at %s", endpoint)
}

// RestartCoreDaemon stops wuji-core on endpoint (if running) and starts a fresh daemon.
// Use after stale driver connections — e.g. wuji-core idle shutdown left a dead llama gRPC link.
func RestartCoreDaemon(ctx context.Context, root, endpoint string) error {
	_ = netx.StopListenerOnEndpoint(endpoint)
	if sock := unixSocketPath(endpoint); sock != "" {
		_ = os.Remove(sock)
	}
	return EnsureCoreDaemon(ctx, root, endpoint)
}

// ResolveCoreBinary locates the wuji-core executable on PATH or next to wuji.
func ResolveCoreBinary(root string) (string, error) {
	if v := strings.TrimSpace(os.Getenv("WUJI_CORE_BIN")); v != "" {
		ok, execErr := executableFile(v)
		if ok {
			return v, nil
		}
		return "", fmt.Errorf("WUJI_CORE_BIN %q: %v", v, execErr)
	}
	for _, candidate := range coreBinaryCandidates(root) {
		if ok, _ := executableFile(candidate); ok {
			return candidate, nil
		}
	}
	if bin, ok := findExecutableInDirs("wuji-core", executableSearchDirs()); ok {
		return bin, nil
	}
	return "", fmt.Errorf("wuji-core not found in PATH (install wuji-core or set WUJI_CORE_BIN)")
}

func coreBinaryCandidates(root string) []string {
	var out []string
	if root != "" && hasWujiStateDir(root) {
		out = append(out, filepath.Join(root, "bin", "wuji-core"))
	}
	if wuji, ok := findExecutableInDirs("wuji", executableSearchDirs()); ok {
		wuji = resolveExecutablePath(wuji)
		out = append(out, filepath.Join(filepath.Dir(wuji), "wuji-core"))
	}
	out = append(out, discoverInstalledCoreBinaries()...)
	return out
}

func resolveExecutablePath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil && resolved != "" {
		return resolved
	}
	return path
}

func executableFile(path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, fmt.Errorf("path is empty")
	}
	path = resolveExecutablePath(path)
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("is a directory")
	}
	if info.Mode()&0o111 == 0 {
		return false, fmt.Errorf("not executable")
	}
	return true, nil
}

func unixSocketPath(endpoint string) string {
	ep := strings.TrimSpace(endpoint)
	const prefix = "unix://"
	if !strings.HasPrefix(ep, prefix) {
		return ""
	}
	return strings.TrimPrefix(ep, prefix)
}

func socketReady(sock string) bool {
	sock = strings.TrimSpace(sock)
	if sock == "" {
		return false
	}
	conn, err := net.DialTimeout("unix", sock, 150*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
