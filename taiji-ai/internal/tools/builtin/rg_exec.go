package builtin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func resolveRipgrepBinary() string {
	if p := strings.TrimSpace(os.Getenv("TAIJI_RG")); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "rg")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if p, err := exec.LookPath("rg"); err == nil {
		return p
	}
	return ""
}

func requireRipgrep() (string, error) {
	bin := resolveRipgrepBinary()
	if bin == "" {
		return "", fmt.Errorf("ripgrep (rg) is required but not found on PATH; install ripgrep or place rg next to the taiji binary")
	}
	return bin, nil
}
