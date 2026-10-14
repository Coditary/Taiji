package wujipath

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const coreModule = "module github.com/coditary/wuji-core"

// ResolveCoreModuleDir returns the filesystem path of the wuji-core Go module (for builds).
func ResolveCoreModuleDir() (string, error) {
	if dir, ok := moduleDirFromEnv(); ok {
		return dir, nil
	}
	if wd, err := os.Getwd(); err == nil {
		if dir, ok := findModuleDir(wd); ok {
			return dir, nil
		}
	}
	if dir, ok := moduleDirFromWujiBinary(); ok {
		return dir, nil
	}
	if dir, ok := moduleDirFromCoreBinary(); ok {
		return dir, nil
	}
	return "", fmt.Errorf("wuji-core module not found (install wuji/wuji-core to PATH or set WUJI_ROOT)")
}

// ResolveRoot picks the Wuji project root for runtime state (.wuji/).
// Runtime resolution uses PATH and explicit config only — never a Taiji checkout symlink.
func ResolveRoot(cfgRoot, configured string) string {
	if v := strings.TrimSpace(os.Getenv("WUJI_ROOT")); v != "" {
		return filepath.Clean(v)
	}
	configured = strings.TrimSpace(configured)
	if configured != "" {
		return resolveRelative(configured, cfgRoot)
	}
	if dir, ok := runtimeRootFromWujiBinary(); ok {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	base := strings.TrimSpace(cfgRoot)
	if base == "" {
		base = "."
	}
	return filepath.Clean(base)
}

func moduleDirFromEnv() (string, bool) {
	for _, key := range []string{"WUJI_ROOT", "WUJI_CORE_MODULE"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if isCoreModuleDir(v) {
				return filepath.Clean(v), true
			}
			if _, err := os.Stat(filepath.Join(v, ".wuji")); err == nil {
				return filepath.Clean(v), true
			}
		}
	}
	return "", false
}

func moduleDirFromWujiBinary() (string, bool) {
	wujiBin, err := exec.LookPath("wuji")
	if err != nil {
		return "", false
	}
	return moduleDirBesideBinary(wujiBin)
}

func moduleDirFromCoreBinary() (string, bool) {
	coreBin, err := exec.LookPath("wuji-core")
	if err != nil {
		return "", false
	}
	return moduleDirBesideBinary(coreBin)
}

func runtimeRootFromWujiBinary() (string, bool) {
	wujiBin, err := exec.LookPath("wuji")
	if err != nil {
		return "", false
	}
	appRoot := filepath.Dir(wujiBin)
	if filepath.Base(appRoot) == "bin" {
		appRoot = filepath.Dir(appRoot)
	}
	if _, err := os.Stat(filepath.Join(appRoot, ".wuji")); err == nil {
		return filepath.Clean(appRoot), true
	}
	if dir, ok := siblingCoreDirs(appRoot); ok {
		return dir, true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(home, ".wuji")); err == nil {
		return home, true
	}
	return "", false
}

func moduleDirBesideBinary(bin string) (string, bool) {
	bin = filepath.Clean(bin)
	dir := filepath.Dir(bin)
	if isCoreModuleDir(dir) {
		return dir, true
	}
	appRoot := dir
	if filepath.Base(dir) == "bin" {
		appRoot = filepath.Dir(dir)
	}
	if isCoreModuleDir(appRoot) {
		return appRoot, true
	}
	return siblingCoreDirs(appRoot)
}

func siblingCoreDirs(appRoot string) (string, bool) {
	parent := filepath.Dir(appRoot)
	candidates := []string{
		filepath.Join(parent, "wuji-core"),
		filepath.Join(parent, "core", "wuji-core"),
		filepath.Join(appRoot, "wuji-core"),
	}
	for _, candidate := range candidates {
		if isCoreModuleDir(candidate) {
			return filepath.Clean(candidate), true
		}
		if _, err := os.Stat(filepath.Join(candidate, ".wuji")); err == nil {
			return filepath.Clean(candidate), true
		}
	}
	return "", false
}

func findModuleDir(start string) (string, bool) {
	dir, err := filepath.Abs(strings.TrimSpace(start))
	if err != nil || dir == "" {
		return "", false
	}
	for {
		if isCoreModuleDir(dir) {
			return filepath.Clean(dir), true
		}
		if isTaijiModule(dir) {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, ".wuji")); err == nil {
			if isWujiAIModule(dir) {
				if sibling, ok := siblingCoreDirs(dir); ok {
					return sibling, true
				}
			}
			return filepath.Clean(dir), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func isCoreModuleDir(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	return err == nil && strings.Contains(string(data), coreModule)
}

func isWujiAIModule(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	return err == nil && strings.Contains(string(data), "module github.com/coditary/wuji-ai")
}

func isTaijiModule(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	return err == nil && strings.Contains(string(data), "module github.com/coditary/taiji")
}

func resolveRelative(root, cfgRoot string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return "."
	}
	if filepath.IsAbs(root) {
		return filepath.Clean(root)
	}
	base := strings.TrimSpace(cfgRoot)
	if base == "" {
		base = "."
	}
	return filepath.Clean(filepath.Join(base, root))
}
