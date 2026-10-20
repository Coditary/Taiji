package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
)

// TurnMetrics is persisted timing/throughput for one chat turn.
type TurnMetrics struct {
	TotalDurationMs      int64   `json:"total_duration_ms,omitempty"`
	WaitDurationMs       int64   `json:"wait_duration_ms,omitempty"`
	ThinkingDurationMs   int64   `json:"thinking_duration_ms,omitempty"`
	GenerationDurationMs int64   `json:"generation_duration_ms,omitempty"`
	OutputTokens         int     `json:"output_tokens,omitempty"`
	InputTokens          int     `json:"input_tokens,omitempty"`
	EstimatedCostUSD     float64 `json:"estimated_cost_usd,omitempty"`
	TokensPerSec         float64 `json:"tokens_per_sec,omitempty"`
	HadThinking          bool    `json:"had_thinking,omitempty"`
	ToolCalls            int     `json:"tool_calls,omitempty"`
	ToolDurationMs       int64   `json:"tool_duration_ms,omitempty"`
	Steps                int     `json:"steps,omitempty"`
	ContextTokens        int     `json:"context_tokens,omitempty"`
	ContextWindow        int     `json:"context_window,omitempty"`
	ContextUsedPct       float64 `json:"context_used_pct,omitempty"`
}

// ChatTurn is a user/assistant pair for TUI history replay.
type ChatTurn struct {
	User       string         `json:"user"`
	Assistant  string         `json:"assistant"`
	UserKind   string         `json:"user_kind,omitempty"`
	UserSentAt time.Time      `json:"user_sent_at,omitempty"`
	Activity   []events.Event `json:"activity,omitempty"`
	Metrics    *TurnMetrics   `json:"metrics,omitempty"`
}

// BuildSidebar assembles the project/session tree for the TUI left sidebar.
func BuildSidebar(currentRoot string, currentLocal bool, currentSessionID string) (config.ProjectsSidebar, error) {
	currentRoot = cleanAbs(currentRoot)
	projects, err := config.VisibleProjects(currentRoot, currentLocal)
	if err != nil {
		return config.ProjectsSidebar{}, err
	}

	out := config.ProjectsSidebar{
		CurrentProject: currentRoot,
		CurrentSession: strings.TrimSpace(currentSessionID),
	}

	for _, project := range projects {
		sessions, err := listSessionsForRoot(project.Root)
		if err != nil {
			continue
		}
		if len(sessions) == 0 {
			continue
		}
		out.Projects = append(out.Projects, config.ProjectSessions{
			Root:     project.Root,
			Name:     project.Name,
			Local:    project.Local,
			Sessions: sessions,
		})
	}
	return out, nil
}

// TurnsFromMessages extracts user/assistant pairs from persisted messages.
func TurnsFromMessages(messages []provider.Message) []ChatTurn {
	var turns []ChatTurn
	var pendingUser string

	for _, msg := range messages {
		switch msg.Role {
		case provider.RoleUser:
			content := strings.TrimSpace(msg.Content)
			if skills.IsContinuationUserMessage(content) || IsHiddenUserMessage(content) {
				continue
			}
			if IsCompactUserMessage(content) {
				summary, _ := ParseCompactUserMessage(content)
				if pendingUser != "" {
					turns = append(turns, ChatTurn{User: pendingUser})
					pendingUser = ""
				}
				turns = append(turns, ChatTurn{User: summary, UserKind: "compact"})
				continue
			}
			if pendingUser != "" {
				turns = append(turns, ChatTurn{User: pendingUser})
			}
			pendingUser = content
		case provider.RoleAssistant:
			content := strings.TrimSpace(msg.Content)
			if pendingUser == "" || content == "" {
				continue
			}
			turns = append(turns, ChatTurn{User: pendingUser, Assistant: content})
			pendingUser = ""
		}
	}
	if pendingUser != "" {
		turns = append(turns, ChatTurn{User: pendingUser})
	}
	return turns
}

func listSessionsForRoot(root string) ([]config.SessionEntry, error) {
	dir := config.SessionsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []config.SessionEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summary, err := loadSummary(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		out = append(out, config.SessionEntry{
			ID:        summary.ID,
			Name:      displaySessionName(summary),
			CreatedAt: summary.CreatedAt,
			Messages:  summary.Messages,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func displaySessionName(summary Summary) string {
	name := strings.TrimSpace(summary.Name)
	if name != "" {
		return name
	}
	if summary.Goal != "" && summary.Goal != "chat" && summary.Goal != "tui-chat" {
		return summary.Goal
	}
	return "Neue Session"
}

func cleanAbs(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

// MarshalSidebarJSON serializes the sidebar payload.
func MarshalSidebarJSON(sidebar config.ProjectsSidebar) (string, error) {
	raw, err := json.Marshal(sidebar)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// MarshalTurnsJSON serializes chat turns for TUI replay.
func MarshalTurnsJSON(turns []ChatTurn) (string, error) {
	raw, err := json.Marshal(turns)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
