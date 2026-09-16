package builtin

import (
	"os"
	"path/filepath"
	"strings"
)

var defaultSkipDirNames = []string{
	"node_modules", ".git", ".wuji", ".taiji", ".cache", "vendor",
}

func explorePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "."
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
			path = filepath.Join(home, path[2:])
		}
	}
	return filepath.Abs(path)
}

func isSkippedPath(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, skip := range defaultSkipDirNames {
		if name == skip {
			return true
		}
	}
	return false
}

func pathHasSkippedSegment(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if isSkippedPath(part) {
			return true
		}
	}
	return false
}

func matchGlob(relPath, pattern string) (bool, error) {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	relPath = filepath.ToSlash(relPath)
	if pattern == "" {
		return false, nil
	}
	if strings.Contains(pattern, "**") {
		suffix := strings.TrimPrefix(pattern, "**/")
		if suffix != pattern {
			if ok, err := filepath.Match(suffix, filepath.Base(relPath)); err != nil {
				return false, err
			} else if ok {
				return true, nil
			}
			return filepath.Match(suffix, relPath)
		}
	}
	if ok, err := filepath.Match(pattern, relPath); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	return filepath.Match(pattern, filepath.Base(relPath))
}
