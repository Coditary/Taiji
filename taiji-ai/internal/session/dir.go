package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/coditary/taiji/internal/config"
)

// DirBinding maps a working directory to the active chat session id.
type DirBinding struct {
	WorkDir   string    `json:"work_dir"`
	SessionID string    `json:"session_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

func dirBindingsDir(root string) string {
	return filepath.Join(config.RuntimeDir(root), "dir-sessions")
}

func dirKey(workDir string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(workDir))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(sum[:16]), nil
}

func bindingPath(root, workDir string) (string, error) {
	key, err := dirKey(workDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dirBindingsDir(root), key+".json"), nil
}

// LoadDirBinding returns the session binding for workDir, if any.
func LoadDirBinding(root, workDir string) (DirBinding, bool, error) {
	path, err := bindingPath(root, workDir)
	if err != nil {
		return DirBinding{}, false, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DirBinding{}, false, nil
	}
	if err != nil {
		return DirBinding{}, false, err
	}
	var binding DirBinding
	if err := json.Unmarshal(raw, &binding); err != nil {
		return DirBinding{}, false, fmt.Errorf("parse dir session binding: %w", err)
	}
	return binding, true, nil
}

// SaveDirBinding records workDir -> sessionID under the Taiji runtime root.
func SaveDirBinding(root, workDir, sessionID string) error {
	abs, err := filepath.Abs(filepath.Clean(workDir))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirBindingsDir(root), 0o755); err != nil {
		return err
	}
	path, err := bindingPath(root, workDir)
	if err != nil {
		return err
	}
	binding := DirBinding{
		WorkDir:   abs,
		SessionID: sessionID,
		UpdatedAt: time.Now().UTC(),
	}
	raw, err := json.MarshalIndent(binding, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
