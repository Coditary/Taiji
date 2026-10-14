package wujipath

import (
	"os"
	"path/filepath"
	"strings"
)

func executableSearchDirs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		out = append(out, dir)
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		add(dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return out
	}
	for _, dir := range []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "go", "bin"),
		filepath.Join(home, "bin"),
	} {
		add(dir)
	}
	if wuji, ok := findExecutableInDirs("wuji", out); ok {
		add(filepath.Dir(wuji))
	}
	return out
}

func findExecutableInDirs(name string, dirs []string) (string, bool) {
	for _, dir := range dirs {
		candidate := filepath.Join(dir, name)
		if ok, _ := executableFile(candidate); ok {
			return candidate, true
		}
	}
	return "", false
}

func discoverInstalledCoreBinaries() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var out []string
	patterns := []string{
		filepath.Join(home, "Dev", "Coditary", "core", "wuji-core", "bin", "wuji-core"),
		filepath.Join(home, "Dev", "*", "core", "wuji-core", "bin", "wuji-core"),
		filepath.Join(home, "Dev", "*", "*", "core", "wuji-core", "bin", "wuji-core"),
		filepath.Join(home, ".local", "share", "wuji", "*", "bin", "wuji-core"),
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		out = append(out, matches...)
	}
	return out
}

func hasWujiStateDir(root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(root, ".wuji"))
	return err == nil
}
