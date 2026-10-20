package agent

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var pathLockMap sync.Map

func lockFilePaths(paths []string) func() {
	if len(paths) == 0 {
		return func() {}
	}
	seen := map[string]bool{}
	unique := make([]string, 0, len(paths))
	for _, p := range paths {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		unique = append(unique, p)
	}
	sort.Strings(unique)

	locks := make([]*sync.Mutex, 0, len(unique))
	for _, p := range unique {
		mu := pathMutex(p)
		mu.Lock()
		locks = append(locks, mu)
	}
	return func() {
		for i := len(locks) - 1; i >= 0; i-- {
			locks[i].Unlock()
		}
	}
}

func pathMutex(path string) *sync.Mutex {
	actual, _ := pathLockMap.LoadOrStore(path, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

func filePathsForToolLock(toolName, argsJSON string) []string {
	switch strings.TrimSpace(toolName) {
	case "write_file", "edit_file", "manage_file":
		argsJSON = strings.TrimSpace(argsJSON)
		if argsJSON == "" {
			return nil
		}
		var args map[string]any
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return nil
		}
		if fromFiles := batchEditPaths(argsJSON); len(fromFiles) > 0 {
			return fromFiles
		}
		var paths []string
		if path, ok := args["path"].(string); ok && strings.TrimSpace(path) != "" {
			paths = append(paths, strings.TrimSpace(path))
		}
		if dest, ok := args["dest"].(string); ok && strings.TrimSpace(dest) != "" {
			paths = append(paths, strings.TrimSpace(dest))
		}
		return paths
	case "batch_edit":
		return batchEditPaths(argsJSON)
	default:
		return nil
	}
}
