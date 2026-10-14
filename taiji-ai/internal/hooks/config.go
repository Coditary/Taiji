package hooks

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// File is the on-disk hooks document (.taiji/hooks.yaml).
type File struct {
	Version int         `yaml:"version"`
	Hooks   []Entry     `yaml:"hooks"`
	Disable []string    `yaml:"disable,omitempty"`
}

// Entry is a single hook definition.
type Entry struct {
	ID         string       `yaml:"id"`
	Event      string       `yaml:"event"`
	Disable    bool         `yaml:"disable,omitempty"`
	Async      bool         `yaml:"async,omitempty"`
	Action     string       `yaml:"action,omitempty"` // "deny" blocks on tool.before when action runs
	Matcher    string       `yaml:"matcher,omitempty"`
	Conditions []Condition  `yaml:"conditions,omitempty"`
	Actions    []ActionSpec `yaml:"actions,omitempty"`
}

// Condition filters when a hook runs.
type Condition struct {
	MatchesAnyPath string `yaml:"matchesAnyPath,omitempty"`
	Tool           string `yaml:"tool,omitempty"`
}

// ActionSpec runs a shell command, bash script path, or in-process plugin.
type ActionSpec struct {
	Command string `yaml:"command,omitempty"`
	Bash    string `yaml:"bash,omitempty"`
	Plugin  string `yaml:"plugin,omitempty"`
	Timeout int    `yaml:"timeout,omitempty"` // seconds
}

// LoadFiles merges hook entries from paths (later paths override by id).
func LoadFiles(paths []string) (*Runner, error) {
	disabled := map[string]struct{}{}
	byID := map[string]Entry{}

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		var doc File
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		for _, id := range doc.Disable {
			disabled[strings.TrimSpace(id)] = struct{}{}
		}
		for _, entry := range doc.Hooks {
			id := strings.TrimSpace(entry.ID)
			if id == "" {
				continue
			}
			if entry.Disable {
				disabled[id] = struct{}{}
				delete(byID, id)
				continue
			}
			if _, off := disabled[id]; off {
				continue
			}
			byID[id] = entry
		}
	}

	entries := make([]Entry, 0, len(byID))
	for _, entry := range byID {
		entries = append(entries, entry)
	}
	return NewRunner(entries), nil
}

// DefaultSearchPaths returns project and global hook file paths.
func DefaultSearchPaths(projectRoot string) []string {
	var paths []string
	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot != "" {
		paths = append(paths, filepath.Join(projectRoot, ".taiji", "hooks.yaml"))
		paths = append(paths, filepath.Join(projectRoot, ".taiji", "hooks.json"))
	}
	if global := globalConfigDir(); global != "" {
		paths = append(paths, filepath.Join(global, "hooks.yaml"))
		paths = append(paths, filepath.Join(global, "hooks.json"))
	}
	return paths
}

func globalConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, ".taiji")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".taiji")
}
