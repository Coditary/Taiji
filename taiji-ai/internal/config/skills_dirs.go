package config

import (
	"os"
	"path/filepath"
	"strings"
)

const skillsSubdir = "skills"

// GlobalSkillsDir returns the user-level skills directory (~/.config/taiji/skills).
func GlobalSkillsDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "taiji", skillsSubdir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "taiji", skillsSubdir)
}

// ProjectSkillsDir returns .taiji/skills under the Taiji project root.
func ProjectSkillsDir(root string) string {
	root = filepath.Clean(root)
	if root == "" || root == "." {
		return filepath.Join(dirName, skillsSubdir)
	}
	return filepath.Join(root, dirName, skillsSubdir)
}

// DefaultSkillsDirs returns skill search paths: project-local first, then global, then bundled.
func DefaultSkillsDirs(root string) []string {
	var out []string
	if dir := ProjectSkillsDir(root); dir != "" {
		out = append(out, dir)
	}
	if dir := GlobalSkillsDir(); dir != "" {
		out = append(out, dir)
	}
	if dir := BundledSkillsDir(root); dir != "" {
		out = append(out, dir)
	}
	return out
}

// BundledSkillsDir returns taiji-ai/global/skills when present (shipped skills).
func BundledSkillsDir(projectRoot string) string {
	projectRoot = strings.TrimSpace(projectRoot)
	candidates := []string{}
	if projectRoot != "" {
		candidates = append(candidates,
			filepath.Join(projectRoot, "taiji-ai", "global", "skills"),
			filepath.Join(projectRoot, "global", "skills"),
		)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "global", "skills"))
	}
	for _, dir := range candidates {
		if skillsDirExists(dir) {
			return dir
		}
	}
	return ""
}

func skillsDirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func ensureSkillsDirs(cfg *Config) {
	if cfg == nil || len(cfg.Skills.Dirs) > 0 {
		return
	}
	cfg.Skills.Dirs = DefaultSkillsDirs(cfg.Root)
}
