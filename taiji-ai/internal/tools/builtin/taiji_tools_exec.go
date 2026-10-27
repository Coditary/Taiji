package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const taijiToolsTimeout = 2 * time.Minute

func resolveTaijiToolsBinary() string {
	if p := strings.TrimSpace(os.Getenv("TAIJI_TOOLS")); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Legacy env/binary name.
	if p := strings.TrimSpace(os.Getenv("TAIJI_GLOBWALK")); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, name := range []string{"taiji-tools", "taiji-globwalk"} {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	if p, err := exec.LookPath("taiji-tools"); err == nil {
		return p
	}
	if p, err := exec.LookPath("taiji-globwalk"); err == nil {
		return p
	}
	for _, candidate := range devTaijiToolsCandidates() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func devTaijiToolsCandidates() []string {
	var paths []string
	if root := workspaceProjectRoot(); root != "" {
		paths = append(paths,
			filepath.Join(root, "taiji-tools", "target", "release", "taiji-tools"),
			filepath.Join(root, "globwalk", "target", "release", "taiji-globwalk"),
		)
	}
	wd, err := os.Getwd()
	if err != nil {
		return paths
	}
	for dir := wd; len(dir) > 1; dir = filepath.Dir(dir) {
		paths = append(paths,
			filepath.Join(dir, "taiji-tools", "target", "release", "taiji-tools"),
			filepath.Join(dir, "globwalk", "target", "release", "taiji-globwalk"),
		)
	}
	return paths
}

func runTaijiTools(ctx context.Context, args ...string) (string, error) {
	bin := resolveTaijiToolsBinary()
	if bin == "" {
		return "", fmt.Errorf("taiji-tools not found (build taiji-ai/taiji-tools or set TAIJI_TOOLS)")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, taijiToolsTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("taiji-tools: %s", msg)
	}
	return stdout.String(), nil
}

func runTaijiToolsStdin(ctx context.Context, stdin string, args ...string) (string, error) {
	bin := resolveTaijiToolsBinary()
	if bin == "" {
		return "", fmt.Errorf("taiji-tools not found (build taiji-ai/taiji-tools or set TAIJI_TOOLS)")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, taijiToolsTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("taiji-tools: %s", msg)
	}
	return stdout.String(), nil
}

func htmlToTextViaTaijiTools(ctx context.Context, html string) (string, error) {
	return runTaijiToolsStdin(ctx, html, "html-to-text")
}

func runTaijiToolsLines(ctx context.Context, args ...string) ([]string, error) {
	out, err := runTaijiTools(ctx, args...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func runGlobwalkBinary(ctx context.Context, root, pattern string, maxResults int) ([]string, error) {
	bin := resolveTaijiToolsBinary()
	if bin != "" && strings.HasSuffix(filepath.Base(bin), "taiji-tools") {
		return runTaijiToolsLines(ctx, "glob", "--root", root, "--pattern", pattern, "--max", fmt.Sprintf("%d", maxResults))
	}
	// Legacy taiji-globwalk flags.
	if bin == "" {
		return nil, fmt.Errorf("taiji-tools not found (build taiji-ai/taiji-tools or set TAIJI_TOOLS)")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, taijiToolsTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, bin, "--root", root, "--pattern", pattern, "--max", fmt.Sprintf("%d", maxResults))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("globwalk: %s", msg)
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
