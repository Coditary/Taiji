package graph

import (
	"os"
	"path/filepath"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// GraphDirs returns search paths for graph blueprints (later overrides same id).
func GraphDirs(projectRoot string) []string {
	seen := map[string]struct{}{}
	var dirs []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		dirs = append(dirs, path)
	}
	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot != "" {
		add(filepath.Join(taijicfg.RuntimeDir(projectRoot), "graphs"))
		add(filepath.Join(projectRoot, "taiji-ai", "global", "graphs"))
		add(filepath.Join(projectRoot, "global", "graphs"))
	}
	if global := taijicfg.GlobalConfigDir(); global != "" {
		add(filepath.Join(global, "graphs"))
	}
	if bundled := taijicfg.BundledTemplatesDir(projectRoot); bundled != "" {
		add(filepath.Join(filepath.Dir(bundled), "graphs"))
	}
	return dirs
}

// ResolveBlueprintPath finds <graphID>/blueprint.yaml under GraphDirs.
func ResolveBlueprintPath(projectRoot, graphID string) (string, error) {
	graphID = strings.TrimSpace(graphID)
	if graphID == "" {
		return "", os.ErrNotExist
	}
	for _, dir := range GraphDirs(projectRoot) {
		path := filepath.Join(dir, graphID, "blueprint.yaml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

// BlueprintDirForWrite returns project-local graph dir for new/edited blueprints.
func BlueprintDirForWrite(projectRoot, graphID string) string {
	return filepath.Join(taijicfg.RuntimeDir(projectRoot), "graphs", strings.TrimSpace(graphID))
}
