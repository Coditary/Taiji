package session

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/config"
)

const checkpointManifestVersion = 1

type checkpointManifest struct {
	Version    int       `json:"version"`
	ExportedAt time.Time `json:"exported_at"`
	SessionID  string    `json:"session_id"`
	Label      string    `json:"label,omitempty"`
}

// ExportCheckpoint writes a portable .taiji.zip archive with full session state.
func ExportCheckpoint(s *Session, destPath string, label string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("session is nil")
	}
	destPath = strings.TrimSpace(destPath)
	if destPath == "" {
		return "", fmt.Errorf("destination path is required")
	}
	if !strings.HasSuffix(strings.ToLower(destPath), ".taiji.zip") && !strings.HasSuffix(strings.ToLower(destPath), ".zip") {
		destPath += ".taiji.zip"
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", err
	}
	tmp := destPath + ".part"
	_ = os.Remove(tmp)
	zf, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	zw := zip.NewWriter(zf)
	manifest := checkpointManifest{
		Version:    checkpointManifestVersion,
		ExportedAt: time.Now().UTC(),
		SessionID:  s.State.ID,
		Label:      strings.TrimSpace(label),
	}
	if err := writeZipJSON(zw, "manifest.json", manifest); err != nil {
		_ = zw.Close()
		_ = zf.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := writeZipJSON(zw, "state.json", s.State); err != nil {
		_ = zw.Close()
		_ = zf.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := zw.Close(); err != nil {
		_ = zf.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := zf.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, destPath); err != nil {
		return "", err
	}
	return destPath, nil
}

// ImportCheckpoint creates a new session from a .taiji.zip archive.
func ImportCheckpoint(projectRoot, archivePath string) (*Session, error) {
	archivePath = strings.TrimSpace(archivePath)
	if archivePath == "" {
		return nil, fmt.Errorf("archive path is required")
	}
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var stateRaw []byte
	var manifest checkpointManifest
	for _, f := range zr.File {
		switch filepath.Base(f.Name) {
		case "manifest.json":
			if err := readZipJSON(f, &manifest); err != nil {
				return nil, err
			}
		case "state.json":
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			stateRaw, err = io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, err
			}
		}
	}
	if len(stateRaw) == 0 {
		return nil, fmt.Errorf("state.json missing from archive")
	}
	var state State
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		return nil, err
	}
	newID := fmt.Sprintf("%d", time.Now().UnixNano())
	dir := filepath.Join(config.SessionsDir(projectRoot), newID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	state.ID = newID
	state.CreatedAt = time.Now().UTC()
	s := &Session{
		Dir:    dir,
		State:  state,
		Events: nil,
	}
	if err := s.Save(); err != nil {
		return nil, err
	}
	_ = manifest
	return s, nil
}

func writeZipJSON(zw *zip.Writer, name string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

func readZipJSON(f *zip.File, v any) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}
