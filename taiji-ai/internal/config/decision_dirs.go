package config

import (
	"os"
	"path/filepath"
	"strings"
)

const decisionsSubdir = "decisions"

// GlobalDecisionsDir returns the user-level saved decisions directory.
func GlobalDecisionsDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "taiji", decisionsSubdir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "taiji", decisionsSubdir)
}

// ProjectDecisionsDir returns .taiji/decisions under the Taiji project root.
func ProjectDecisionsDir(root string) string {
	root = filepath.Clean(root)
	if root == "" || root == "." {
		return filepath.Join(dirName, decisionsSubdir)
	}
	return filepath.Join(root, dirName, decisionsSubdir)
}

// DefaultDecisionProfileDirs returns profile search paths: project, global, bundled.
func DefaultDecisionProfileDirs(root string) []string {
	var out []string
	if dir := ProjectDecisionsDir(root); dir != "" {
		out = append(out, dir)
	}
	if dir := GlobalDecisionsDir(); dir != "" {
		out = append(out, dir)
	}
	if dir := BundledDecisionsDir(root); dir != "" {
		out = append(out, dir)
	}
	return out
}

// BundledDecisionsDir returns taiji-ai/global/decisions when present.
func BundledDecisionsDir(projectRoot string) string {
	projectRoot = strings.TrimSpace(projectRoot)
	candidates := []string{}
	if projectRoot != "" {
		candidates = append(candidates,
			filepath.Join(projectRoot, "taiji-ai", "global", decisionsSubdir),
			filepath.Join(projectRoot, "global", decisionsSubdir),
		)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "global", decisionsSubdir))
	}
	for _, dir := range candidates {
		if decisionsDirExists(dir) {
			return dir
		}
	}
	return ""
}

func decisionsDirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func ensureDecisionProfileDirs(cfg *Config) {
	if cfg == nil {
		return
	}
	if len(cfg.Decisions.Dirs) > 0 {
		return
	}
	cfg.Decisions.Dirs = DefaultDecisionProfileDirs(cfg.Root)
}
