package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coditary/taiji/internal/config"
)

// Delete removes a session directory and any directory bindings that reference it.
func Delete(root, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.Contains(id, string(os.PathSeparator)) || strings.Contains(id, "..") {
		return fmt.Errorf("invalid session id %q", id)
	}

	dir := filepath.Join(config.SessionsDir(root), id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("session %q not found", id)
	} else if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return clearBindingsForSession(root, id)
}

func clearBindingsForSession(root, sessionID string) error {
	bindingsDir := dirBindingsDir(root)
	entries, err := os.ReadDir(bindingsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(bindingsDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var binding DirBinding
		if err := json.Unmarshal(raw, &binding); err != nil {
			continue
		}
		if binding.SessionID != sessionID {
			continue
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}
