package hooks

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadFromPaths loads YAML and Claude-compatible JSON hook files.
func LoadFromPaths(paths []string) (*Runner, error) {
	disabled := map[string]struct{}{}
	byID := map[string]Entry{}

	var yamlPaths []string
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if strings.EqualFold(filepath.Ext(path), ".json") {
			entries, err := LoadClaudeJSON(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			for _, entry := range entries {
				if entry.ID == "" {
					continue
				}
				byID[entry.ID] = entry
			}
			continue
		}
		yamlPaths = append(yamlPaths, path)
	}

	if len(yamlPaths) > 0 {
		runner, err := LoadFiles(yamlPaths)
		if err != nil {
			return nil, err
		}
		if runner != nil {
			for _, entry := range runner.entries {
				if entry.Disable {
					disabled[entry.ID] = struct{}{}
					delete(byID, entry.ID)
					continue
				}
				if _, off := disabled[entry.ID]; off {
					continue
				}
				byID[entry.ID] = entry
			}
		}
	}

	if len(byID) == 0 {
		return nil, nil
	}
	entries := make([]Entry, 0, len(byID))
	for _, entry := range byID {
		entries = append(entries, entry)
	}
	return NewRunner(entries), nil
}
