package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type todoStatus string

const (
	todoPending    todoStatus = "pending"
	todoInProgress todoStatus = "in_progress"
	todoDone       todoStatus = "done"
	todoCancelled  todoStatus = "cancelled"
)

type todoItem struct {
	ID     string     `json:"id"`
	Task   string     `json:"task"`
	Status todoStatus `json:"status"`
}

// TodoManager stores per-session todo lists.
type TodoManager struct {
	mu       sync.Mutex
	sessions map[string][]todoItem
}

var defaultTodos = NewTodoManager()

// DefaultTodoManager returns the shared todo manager.
func DefaultTodoManager() *TodoManager {
	return defaultTodos
}

// NewTodoManager creates an empty todo manager.
func NewTodoManager() *TodoManager {
	return &TodoManager{sessions: map[string][]todoItem{}}
}

func (m *TodoManager) Set(sessionID string, items []todoItem) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("items must not be empty for set")
	}
	m.mu.Lock()
	m.sessions[normalizeSessionID(sessionID)] = items
	m.mu.Unlock()
	InvalidateSessionHUDCache(sessionID)
	if err := m.persist(sessionID, items); err != nil {
		return "", err
	}
	return m.FormatList(sessionID), nil
}

func (m *TodoManager) Update(sessionID string, items []todoItem) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("items must not be empty for update")
	}
	sid := normalizeSessionID(sessionID)
	m.mu.Lock()
	current := m.sessions[sid]
	if current == nil {
		current = m.loadFromDisk(sessionID)
	}
	byID := map[string]todoItem{}
	for _, it := range current {
		byID[it.ID] = it
	}
	for _, upd := range items {
		if strings.TrimSpace(upd.ID) == "" {
			continue
		}
		prev := byID[upd.ID]
		if strings.TrimSpace(upd.Task) != "" {
			prev.Task = strings.TrimSpace(upd.Task)
		}
		if upd.Status != "" {
			prev.Status = upd.Status
		}
		prev.ID = upd.ID
		byID[upd.ID] = prev
	}
	merged := make([]todoItem, 0, len(byID))
	for _, it := range byID {
		merged = append(merged, it)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].ID < merged[j].ID })
	m.sessions[sid] = merged
	m.mu.Unlock()
	InvalidateSessionHUDCache(sessionID)
	if err := m.persist(sessionID, merged); err != nil {
		return "", err
	}
	return m.FormatList(sessionID), nil
}

func (m *TodoManager) Get(sessionID string) string {
	return m.FormatList(sessionID)
}

func (m *TodoManager) FormatList(sessionID string) string {
	items := m.items(sessionID)
	if len(items) == 0 {
		return "no todos"
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	var lines []string
	for _, it := range items {
		lines = append(lines, formatTodoLine(it))
	}
	return strings.Join(lines, "\n")
}

func (m *TodoManager) OpenReminder(sessionID string) string {
	items := m.items(sessionID)
	var open []todoItem
	for _, it := range items {
		if it.Status == todoPending || it.Status == todoInProgress {
			open = append(open, it)
		}
	}
	if len(open) == 0 {
		return ""
	}
	sort.Slice(open, func(i, j int) bool { return open[i].ID < open[j].ID })
	var lines []string
	lines = append(lines, "### Session todo plan (open items)")
	for _, it := range open {
		lines = append(lines, formatTodoLine(it))
	}
	lines = append(lines, "Update progress with todo_plan (update) when a step completes.")
	return strings.Join(lines, "\n")
}

func formatTodoLine(it todoItem) string {
	mark := " "
	switch it.Status {
	case todoDone:
		mark = "x"
	case todoInProgress:
		mark = ">"
	case todoCancelled:
		mark = "-"
	}
	return fmt.Sprintf("[%s] %s. %s", mark, it.ID, strings.TrimSpace(it.Task))
}

func (m *TodoManager) items(sessionID string) []todoItem {
	sid := normalizeSessionID(sessionID)
	m.mu.Lock()
	items := m.sessions[sid]
	m.mu.Unlock()
	if items != nil {
		return append([]todoItem(nil), items...)
	}
	return m.loadFromDisk(sessionID)
}

func (m *TodoManager) persist(sessionID string, items []todoItem) error {
	root := workspaceSessionsRoot()
	if root == "" {
		return nil
	}
	dir := filepath.Join(root, normalizeSessionID(sessionID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "todos.json"), raw, 0o644)
}

func (m *TodoManager) loadFromDisk(sessionID string) []todoItem {
	root := workspaceSessionsRoot()
	if root == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(root, normalizeSessionID(sessionID), "todos.json"))
	if err != nil {
		return nil
	}
	var items []todoItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	sid := normalizeSessionID(sessionID)
	m.mu.Lock()
	m.sessions[sid] = items
	m.mu.Unlock()
	return items
}

func normalizeSessionID(id string) string {
	if strings.TrimSpace(id) == "" {
		return "default"
	}
	return strings.TrimSpace(id)
}

func parseTodoItems(args map[string]any) ([]todoItem, error) {
	raw, ok := args["items"]
	if !ok {
		return nil, fmt.Errorf("items is required")
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("items must be an array")
	}
	out := make([]todoItem, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("item %d must be an object", i+1)
		}
		id := strings.TrimSpace(stringArg(m, "id"))
		task := strings.TrimSpace(stringArg(m, "task"))
		status := todoStatus(strings.TrimSpace(stringArg(m, "status")))
		if id == "" {
			return nil, fmt.Errorf("item %d: id is required", i+1)
		}
		if task == "" && status == "" {
			return nil, fmt.Errorf("item %d: task or status required", i+1)
		}
		if status == "" {
			status = todoPending
		}
		out = append(out, todoItem{ID: id, Task: task, Status: status})
	}
	return out, nil
}
