package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coditary/taiji/internal/diff"
	"github.com/coditary/taiji/internal/rag"
)

type fileBaseline struct {
	Existed bool
	Content string
}

type checkpointMeta struct {
	ID           string    `json:"id"`
	Label        string    `json:"label"`
	Created      time.Time `json:"created"`
	Files        []string  `json:"files"`
	CreatedFiles []string  `json:"created_files,omitempty"`
}

type sessionWorkspace struct {
	mu            sync.Mutex
	sessionID     string
	baselines     map[string]fileBaseline
	modifiedPaths map[string]bool
}

// WorkspaceManager tracks per-session file baselines and checkpoints.
type WorkspaceManager struct {
	mu                sync.Mutex
	sessions          map[string]*sessionWorkspace
	seq               uint64
	latestCheckpoints map[string]checkpointMeta
}

var defaultWorkspace = NewWorkspaceManager()

// DefaultWorkspace returns the shared workspace manager.
func DefaultWorkspace() *WorkspaceManager {
	return defaultWorkspace
}

// NewWorkspaceManager creates an empty workspace manager.
func NewWorkspaceManager() *WorkspaceManager {
	return &WorkspaceManager{
		sessions:          map[string]*sessionWorkspace{},
		latestCheckpoints: map[string]checkpointMeta{},
	}
}

func (m *WorkspaceManager) session(id string) *sessionWorkspace {
	if strings.TrimSpace(id) == "" {
		id = "default"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sw, ok := m.sessions[id]
	if !ok {
		sw = &sessionWorkspace{
			sessionID:     id,
			baselines:     map[string]fileBaseline{},
			modifiedPaths: map[string]bool{},
		}
		m.sessions[id] = sw
	}
	return sw
}

// CapturePreWrite records the original file state before the first modification in a session.
func (m *WorkspaceManager) CapturePreWrite(_ context.Context, sessionID, path string) error {
	path, err := canonicalWorkspacePath(path)
	if err != nil {
		return err
	}
	sw := m.session(sessionID)
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if _, ok := sw.baselines[path]; ok {
		return nil
	}
	bl := fileBaseline{Existed: false}
	if raw, err := os.ReadFile(path); err == nil {
		bl.Existed = true
		bl.Content = string(raw)
	} else if !os.IsNotExist(err) {
		return err
	}
	sw.baselines[path] = bl
	return nil
}

func (m *WorkspaceManager) noteModified(sessionID, path string) {
	path, err := canonicalWorkspacePath(path)
	if err != nil {
		return
	}
	sw := m.session(sessionID)
	sw.mu.Lock()
	sw.modifiedPaths[path] = true
	sw.mu.Unlock()
}

func (m *WorkspaceManager) CreateCheckpoint(sessionID, label string) (string, string, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		label = "checkpoint"
	}
	sw := m.session(sessionID)
	sw.mu.Lock()
	paths := sw.modifiedPathList()
	sw.mu.Unlock()
	if len(paths) == 0 {
		return "", "", fmt.Errorf("no session file changes to checkpoint yet")
	}

	m.mu.Lock()
	m.seq++
	id := fmt.Sprintf("cp-%d", m.seq)
	m.mu.Unlock()

	files := map[string]string{}
	var created []string
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", "", err
		}
		files[path] = string(raw)
		sw.mu.Lock()
		bl := sw.baselines[path]
		sw.mu.Unlock()
		if !bl.Existed {
			created = append(created, path)
		}
	}
	if err := m.saveCheckpoint(sessionID, id, label, files, created); err != nil {
		return "", "", err
	}
	m.rememberLatestCheckpoint(sessionID, checkpointMeta{
		ID: id, Label: label, Created: time.Now().UTC(), Files: sortedKeys(files), CreatedFiles: created,
	})
	InvalidateSessionHUDCache(sessionID)
	git := gitStatusShort()
	summary := fmt.Sprintf("checkpoint %s (%q) saved %d file(s)", id, label, len(files))
	if git != "" {
		summary += "\n\ngit status --short:\n" + git
	}
	return id, summary, nil
}

func (m *WorkspaceManager) ListCheckpoints(sessionID string) (string, error) {
	dir := checkpointsRoot(sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "no checkpoints", nil
		}
		return "", err
	}
	var lines []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := loadCheckpointMeta(sessionID, e.Name())
		if err != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s — %q — %d file(s) — %s",
			meta.ID, meta.Label, len(meta.Files), meta.Created.Format(time.RFC3339)))
	}
	if len(lines) == 0 {
		return "no checkpoints", nil
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

func (m *WorkspaceManager) Diff(sessionID, checkpointID, singlePath string) (string, error) {
	singlePath = strings.TrimSpace(singlePath)
	if strings.TrimSpace(checkpointID) == "" {
		return gitDiffHEAD(singlePath)
	}
	cpFiles, err := m.loadCheckpointFiles(sessionID, checkpointID)
	if err != nil {
		return "", err
	}
	var paths []string
	if singlePath != "" {
		p, err := canonicalWorkspacePath(singlePath)
		if err != nil {
			return "", err
		}
		paths = []string{p}
	} else {
		paths = sortedKeys(cpFiles)
	}
	var blocks []string
	for _, path := range paths {
		old := cpFiles[path]
		newBytes, err := os.ReadFile(path)
		newText := ""
		if err == nil {
			newText = string(newBytes)
		} else if !os.IsNotExist(err) {
			return "", err
		}
		u := diff.Unified(path, old, newText, 30)
		if u == "" {
			continue
		}
		blocks = append(blocks, u)
	}
	if len(blocks) == 0 {
		return "no differences", nil
	}
	return strings.Join(blocks, "\n\n"), nil
}

func (m *WorkspaceManager) Restore(ctx context.Context, sessionID, checkpointID string, paths []string) (string, error) {
	cpFiles, err := m.loadCheckpointFiles(sessionID, checkpointID)
	if err != nil {
		return "", err
	}
	targets := cpFiles
	if len(paths) > 0 {
		filtered := map[string]string{}
		for _, p := range paths {
			canon, err := canonicalWorkspacePath(p)
			if err != nil {
				return "", err
			}
			if content, ok := cpFiles[canon]; ok {
				filtered[canon] = content
			}
		}
		targets = filtered
	}
	var restored []string
	for path, content := range targets {
		if err := writeFileAtomic(path, content); err != nil {
			return "", err
		}
		restored = append(restored, path)
		if svc := workspaceRAG(); svc != nil && rag.ShouldAutoIndexPath(path) {
			_, _ = rag.AutoIndexFile(ctx, svc, path)
		}
	}
	if len(paths) == 0 {
		sw := m.session(sessionID)
		sw.mu.Lock()
		for path := range sw.modifiedPaths {
			if _, inCP := cpFiles[path]; inCP {
				continue
			}
			bl := sw.baselines[path]
			if !bl.Existed {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					sw.mu.Unlock()
					return "", err
				}
			}
		}
		sw.mu.Unlock()
	}
	sort.Strings(restored)
	return fmt.Sprintf("restored checkpoint %s (%d file(s))", checkpointID, len(restored)), nil
}

func (sw *sessionWorkspace) modifiedPathList() []string {
	out := make([]string, 0, len(sw.modifiedPaths))
	for p := range sw.modifiedPaths {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func checkpointsRoot(sessionID string) string {
	root := workspaceSessionsRoot()
	if root == "" {
		root = os.TempDir()
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "default"
	}
	return filepath.Join(root, sessionID, "checkpoints")
}

func (m *WorkspaceManager) saveCheckpoint(sessionID, id, label string, files map[string]string, created []string) error {
	dir := filepath.Join(checkpointsRoot(sessionID), id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	meta := checkpointMeta{
		ID:           id,
		Label:        label,
		Created:      time.Now().UTC(),
		Files:        sortedKeys(files),
		CreatedFiles: created,
	}
	for path, content := range files {
		blob := filepath.Join(dir, blobName(path))
		if err := os.WriteFile(blob, []byte(content), 0o644); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "meta.json"), raw, 0o644)
}

func (m *WorkspaceManager) rememberLatestCheckpoint(sessionID string, meta checkpointMeta) {
	sessionID = normalizeSessionID(sessionID)
	m.mu.Lock()
	m.latestCheckpoints[sessionID] = meta
	m.mu.Unlock()
}

// LatestCheckpointMeta returns the most recently created checkpoint for a session (cached).
func (m *WorkspaceManager) LatestCheckpointMeta(sessionID string) *checkpointMeta {
	sessionID = normalizeSessionID(sessionID)
	m.mu.Lock()
	meta, ok := m.latestCheckpoints[sessionID]
	m.mu.Unlock()
	if ok {
		cp := meta
		return &cp
	}
	latest, err := m.loadLatestCheckpointFromDisk(sessionID)
	if err != nil || latest == nil {
		return nil
	}
	m.rememberLatestCheckpoint(sessionID, *latest)
	return latest
}

func (m *WorkspaceManager) loadLatestCheckpointFromDisk(sessionID string) (*checkpointMeta, error) {
	dir := checkpointsRoot(sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var best *checkpointMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := loadCheckpointMeta(sessionID, e.Name())
		if err != nil {
			continue
		}
		if best == nil || meta.Created.After(best.Created) {
			best = meta
		}
	}
	return best, nil
}

func loadCheckpointMeta(sessionID, id string) (*checkpointMeta, error) {
	raw, err := os.ReadFile(filepath.Join(checkpointsRoot(sessionID), id, "meta.json"))
	if err != nil {
		return nil, err
	}
	var meta checkpointMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (m *WorkspaceManager) loadCheckpointFiles(sessionID, id string) (map[string]string, error) {
	meta, err := loadCheckpointMeta(sessionID, id)
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	dir := filepath.Join(checkpointsRoot(sessionID), id)
	for _, path := range meta.Files {
		raw, err := os.ReadFile(filepath.Join(dir, blobName(path)))
		if err != nil {
			return nil, err
		}
		files[path] = string(raw)
	}
	return files, nil
}

func blobName(path string) string {
	return strings.ReplaceAll(path, "/", "__")
}

func canonicalWorkspacePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(workspaceProjectRoot(), path)
	}
	return filepath.Clean(path), nil
}

func writeFileAtomic(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func gitStatusShort() string {
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = workspaceProjectRoot()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitDiffHEAD(singlePath string) (string, error) {
	args := []string{"diff", "HEAD"}
	if strings.TrimSpace(singlePath) != "" {
		args = append(args, singlePath)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = workspaceProjectRoot()
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", fmt.Errorf("git diff: %w", err)
		}
	}
	if text == "" {
		return "no differences (git diff HEAD)", nil
	}
	return text, nil
}
