package builtin

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// SessionHUD is a lightweight snapshot for TUI / embed clients.
type SessionHUD struct {
	TodoSummary      string
	ActiveJobCount   int
	ActiveJobNames   []string
	LatestCheckpoint string
}

const hudCacheTTL = 250 * time.Millisecond

var (
	hudCacheMu sync.Mutex
	hudCache   = map[string]hudCacheEntry{}
)

type hudCacheEntry struct {
	until time.Time
	hud   SessionHUD
}

// GetSessionHUD returns todo, job, and checkpoint status for a chat session.
func GetSessionHUD(sessionID string) SessionHUD {
	sessionID = normalizeSessionID(sessionID)
	now := time.Now()
	hudCacheMu.Lock()
	if ent, ok := hudCache[sessionID]; ok && now.Before(ent.until) {
		hud := ent.hud
		hudCacheMu.Unlock()
		return hud
	}
	hudCacheMu.Unlock()

	hud := buildSessionHUD(sessionID)
	hudCacheMu.Lock()
	hudCache[sessionID] = hudCacheEntry{until: now.Add(hudCacheTTL), hud: hud}
	hudCacheMu.Unlock()
	return hud
}

// InvalidateSessionHUDCache drops cached HUD for a session (tests / after mutations).
func InvalidateSessionHUDCache(sessionID string) {
	sessionID = normalizeSessionID(sessionID)
	hudCacheMu.Lock()
	delete(hudCache, sessionID)
	hudCacheMu.Unlock()
}

func buildSessionHUD(sessionID string) SessionHUD {
	return SessionHUD{
		TodoSummary:      todoHUDSummary(sessionID),
		ActiveJobCount:   len(runningJobs(sessionID)),
		ActiveJobNames:   runningJobs(sessionID),
		LatestCheckpoint: latestCheckpointLabel(sessionID),
	}
}

func todoHUDSummary(sessionID string) string {
	items := DefaultTodoManager().items(sessionID)
	if len(items) == 0 {
		return ""
	}
	done, total := 0, len(items)
	var current string
	for _, it := range items {
		if it.Status == todoDone || it.Status == todoCancelled {
			done++
		}
		if it.Status == todoInProgress && current == "" {
			current = strings.TrimSpace(it.Task)
		}
	}
	if current == "" {
		for _, it := range items {
			if it.Status == todoPending {
				current = strings.TrimSpace(it.Task)
				break
			}
		}
	}
	summary := fmt.Sprintf("Done: %d, Total: %d", done, total)
	if current != "" {
		summary += fmt.Sprintf(", CurrentTask: %q", current)
	}
	return summary
}

func runningJobs(sessionID string) []string {
	jm := DefaultJobManager()
	jm.mu.Lock()
	sj := jm.sessions[normalizeSessionID(sessionID)]
	jm.mu.Unlock()
	if sj == nil {
		return nil
	}
	sj.mu.Lock()
	defer sj.mu.Unlock()
	var out []string
	for id, job := range sj.jobs {
		if job != nil && job.running {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func latestCheckpointLabel(sessionID string) string {
	if meta := DefaultWorkspace().LatestCheckpointMeta(sessionID); meta != nil {
		label := strings.TrimSpace(meta.Label)
		if label == "" {
			return meta.ID
		}
		return fmt.Sprintf("%s (%s)", meta.ID, label)
	}
	return ""
}

// FormatSessionHUDLine renders a compact footer line for the TUI.
func FormatSessionHUDLine(h SessionHUD) string {
	var parts []string
	if s := strings.TrimSpace(h.TodoSummary); s != "" {
		parts = append(parts, "Todos: "+s)
	}
	if h.ActiveJobCount > 0 {
		label := fmt.Sprintf("%d running", h.ActiveJobCount)
		if len(h.ActiveJobNames) > 0 {
			label += " (" + strings.Join(h.ActiveJobNames, ", ") + ")"
		}
		parts = append(parts, "Jobs: "+label)
	}
	if cp := strings.TrimSpace(h.LatestCheckpoint); cp != "" {
		parts = append(parts, "Checkpoint: "+cp)
	}
	return strings.Join(parts, " · ")
}
