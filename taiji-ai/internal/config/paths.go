package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

const legacyGlobalDirName = ".taiji"

// GlobalStateDir returns the user-level Taiji data directory (XDG_DATA_HOME/taiji).
func GlobalStateDir() string {
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		return filepath.Join(xdg, "taiji")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "taiji")
}

// LegacyGlobalConfigDir is the pre-XDG global directory (~/.taiji).
func LegacyGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, legacyGlobalDirName)
}

// ProjectKey returns a stable directory name for a project root under GlobalStateDir/projects/.
func ProjectKey(root string) string {
	root = cleanProjectRoot(root)
	if root == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(root))
	return hex.EncodeToString(sum[:16])
}

// ProjectStorageLocal reports whether runtime data for root lives in <root>/.taiji.
func ProjectStorageLocal(root string) bool {
	root = cleanProjectRoot(root)
	if root == "" {
		return false
	}
	for _, entry := range mustLoadProjectsRegistry() {
		if cleanProjectRoot(entry.Root) == root {
			return entry.Local
		}
	}
	if hasTaijiConfig(root) {
		managed := filepath.Join(ManagedRuntimeDir(root), configFile)
		if !fileExists(managed) {
			return true
		}
	}
	return false
}

func mustLoadProjectsRegistry() []ProjectEntry {
	entries, _ := loadProjectsRegistry()
	return entries
}

// ManagedRuntimeDir returns the global store path for a non-local project.
func ManagedRuntimeDir(root string) string {
	key := ProjectKey(root)
	if key == "" {
		return ""
	}
	return filepath.Join(GlobalStateDir(), "projects", key)
}

// RuntimeDirForRoot returns the Taiji runtime directory for a project root.
func RuntimeDirForRoot(root string, storageLocal bool) string {
	root = filepath.Clean(root)
	if storageLocal || root == "" || root == "." {
		if root == "" || root == "." {
			return dirName
		}
		return filepath.Join(root, dirName)
	}
	return ManagedRuntimeDir(root)
}

// ResolveStorageLocal combines explicit --local with registry / legacy layout.
func ResolveStorageLocal(root string, opts LoadOptions) bool {
	if opts.Local {
		return true
	}
	return ProjectStorageLocal(root)
}

func globalConfigSearchDirs() []string {
	primary := GlobalConfigDir()
	legacy := LegacyGlobalConfigDir()
	var dirs []string
	if primary != "" {
		dirs = append(dirs, primary)
	}
	if legacy != "" && legacy != primary {
		dirs = append(dirs, legacy)
	}
	return dirs
}

func readFirstExistingConfigFile(name string) ([]byte, string, bool) {
	for _, dir := range globalConfigSearchDirs() {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err == nil {
			return raw, path, true
		}
	}
	return nil, "", false
}

func projectConfigPath(root string, storageLocal bool) string {
	return filepath.Join(RuntimeDirForRoot(root, storageLocal), configFile)
}

func resolveProjectConfigPath(root string, storageLocal bool) string {
	primary := projectConfigPath(root, storageLocal)
	if fileExists(primary) {
		return primary
	}
	if !storageLocal {
		legacy := filepath.Join(root, dirName, configFile)
		if fileExists(legacy) {
			return legacy
		}
	}
	return primary
}

func ensureManagedProjectRootMarker(root string) error {
	if root == "" {
		return nil
	}
	dir := ManagedRuntimeDir(root)
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	marker := filepath.Join(dir, "project_root.txt")
	if fileExists(marker) {
		return nil
	}
	return os.WriteFile(marker, []byte(cleanProjectRoot(root)+"\n"), 0o644)
}
