package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const projectsFile = "projects.json"

// ProjectEntry tracks a known Taiji project root for the session sidebar.
type ProjectEntry struct {
	Root         string    `json:"root"`
	Name         string    `json:"name"`
	Local        bool      `json:"local,omitempty"`
	LastAccessed time.Time `json:"last_accessed"`
}

// SessionEntry is a lightweight session row for the sidebar.
type SessionEntry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Messages  int       `json:"messages"`
}

// ProjectSessions groups sessions under one project root.
type ProjectSessions struct {
	Root     string         `json:"root"`
	Name     string         `json:"name"`
	Local    bool           `json:"local,omitempty"`
	Sessions []SessionEntry `json:"sessions"`
}

// ProjectsSidebar is the JSON payload for the TUI left sidebar.
type ProjectsSidebar struct {
	CurrentProject  string            `json:"current_project"`
	CurrentSession  string            `json:"current_session"`
	Projects        []ProjectSessions `json:"projects"`
}

func projectsRegistryPath() string {
	dir := GlobalConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, projectsFile)
}

// RegisterProject records or updates a project root in the global registry.
func RegisterProject(root string, local bool) error {
	root = cleanProjectRoot(root)
	if root == "" {
		return nil
	}
	path := projectsRegistryPath()
	if path == "" {
		return nil
	}

	entries, _ := loadProjectsRegistry()
	found := false
	now := time.Now().UTC()
	for i, entry := range entries {
		if cleanProjectRoot(entry.Root) == root {
			entries[i].LastAccessed = now
			entries[i].Local = local
			if strings.TrimSpace(entries[i].Name) == "" {
				entries[i].Name = projectDisplayName(root)
			}
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, ProjectEntry{
			Root:         root,
			Name:         projectDisplayName(root),
			Local:        local,
			LastAccessed: now,
		})
	}
	return saveProjectsRegistry(entries)
}

func loadProjectsRegistry() ([]ProjectEntry, error) {
	path := projectsRegistryPath()
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []ProjectEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func saveProjectsRegistry(entries []ProjectEntry) error {
	path := projectsRegistryPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastAccessed.After(entries[j].LastAccessed)
	})
	raw, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func cleanProjectRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return abs
}

func projectDisplayName(root string) string {
	root = cleanProjectRoot(root)
	if root == "" {
		return ""
	}
	return filepath.Base(root)
}

// VisibleProjects returns project entries that should appear in the sidebar.
// Local-only projects are visible only when the viewer is in that exact root.
func VisibleProjects(currentRoot string, currentLocal bool) ([]ProjectEntry, error) {
	currentRoot = cleanProjectRoot(currentRoot)
	entries, err := loadProjectsRegistry()
	if err != nil {
		return nil, err
	}

	var visible []ProjectEntry
	seen := map[string]struct{}{}

	add := func(entry ProjectEntry) {
		root := cleanProjectRoot(entry.Root)
		if root == "" {
			return
		}
		if _, ok := seen[root]; ok {
			return
		}
		if !projectHasSessions(root) {
			return
		}
		if entry.Local && root != currentRoot {
			return
		}
		if currentLocal && root != currentRoot {
			return
		}
		seen[root] = struct{}{}
		if strings.TrimSpace(entry.Name) == "" {
			entry.Name = projectDisplayName(root)
		}
		entry.Root = root
		visible = append(visible, entry)
	}

	for _, entry := range entries {
		add(entry)
	}

	if currentRoot != "" && projectHasSessions(currentRoot) {
		add(ProjectEntry{
			Root:  currentRoot,
			Name:  projectDisplayName(currentRoot),
			Local: currentLocal,
		})
	}

	sort.Slice(visible, func(i, j int) bool {
		if visible[i].Root == currentRoot {
			return true
		}
		if visible[j].Root == currentRoot {
			return false
		}
		return visible[i].LastAccessed.After(visible[j].LastAccessed)
	})
	return visible, nil
}

func projectHasSessions(root string) bool {
	dir := SessionsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}
	return false
}
