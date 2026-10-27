package session

import (
	"path/filepath"
	"strings"
)

// ProjectRoot returns the project directory that owns a session on disk (<root>/.taiji/sessions/<id>).
func ProjectRoot(s *Session) string {
	if s == nil {
		return ""
	}
	dir := strings.TrimSpace(s.Dir)
	if dir == "" {
		return ""
	}
	// <root>/.taiji/sessions/<id>
	return cleanAbs(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Clean(dir)))))
}
