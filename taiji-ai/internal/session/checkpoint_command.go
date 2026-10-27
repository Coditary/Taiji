package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/config"
)

// ParseCheckpointCommand handles /checkpoint, /export-session, /import-session.
func ParseCheckpointCommand(s *Session, projectRoot, line string) (reply string, handled bool, err error) {
	if s == nil {
		return "", false, nil
	}
	line = strings.TrimSpace(line)
	lower := strings.ToLower(line)
	fields := strings.Fields(line)

	switch {
	case lower == "/checkpoint" || strings.HasPrefix(lower, "/checkpoint "):
		label := "checkpoint"
		if len(fields) > 1 {
			label = strings.Join(fields[1:], "-")
		}
		dir := filepath.Join(s.Dir, "checkpoints")
		safe := strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
				return r
			}
			return '-'
		}, label)
		dest := filepath.Join(dir, fmt.Sprintf("%s-%s.taiji.zip", safe, time.Now().Format("20060102-150405")))
		path, err := ExportCheckpoint(s, dest, label)
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("Checkpoint gespeichert: %s", path), true, nil

	case lower == "/export-session" || strings.HasPrefix(lower, "/export-session "):
		dest := ""
		if len(fields) > 1 {
			dest = strings.Join(fields[1:], " ")
		}
		if dest == "" {
			dir := filepath.Join(config.SessionsDir(projectRoot), "exports")
			dest = filepath.Join(dir, fmt.Sprintf("%s-%s.taiji.zip", s.State.ID, time.Now().Format("20060102-150405")))
		} else if strings.HasPrefix(dest, "~") {
			home, _ := os.UserHomeDir()
			dest = filepath.Join(home, strings.TrimPrefix(dest, "~/"))
		}
		path, err := ExportCheckpoint(s, dest, "export")
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("Session exportiert: %s", path), true, nil

	case strings.HasPrefix(lower, "/import-session "):
		if len(fields) < 2 {
			return "Nutze: /import-session <pfad.taiji.zip>", true, nil
		}
		path := strings.Join(fields[1:], " ")
		if strings.HasPrefix(path, "~") {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
		imported, err := ImportCheckpoint(projectRoot, path)
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("Session importiert: %s (neue ID %s)", path, imported.State.ID), true, nil

	default:
		return "", false, nil
	}
}
