package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

// State is persisted session metadata.
type State struct {
	ID         string             `json:"id"`
	Name       string             `json:"name,omitempty"`
	NameManual bool               `json:"name_manual,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	Provider   string             `json:"provider"`
	Goal       string             `json:"goal"`
	Messages   []provider.Message `json:"messages"`
	Stats      Stats              `json:"stats,omitempty"`
	Settings   Settings           `json:"settings,omitempty"`
	Graph      *GraphSession      `json:"graph,omitempty"`
}

// Stats aggregates usage for a session.
type Stats struct {
	Requests         int       `json:"requests"`
	InputTokens      int       `json:"input_tokens"`
	OutputTokens     int       `json:"output_tokens"`
	LastInputTokens  int       `json:"last_input_tokens,omitempty"`
	TotalLatencyMs   int64     `json:"total_latency_ms"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd"`
	Steps            int       `json:"steps"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// Session wraps runtime session state.
type Session struct {
	State  State
	Events *events.Writer
	Dir    string
}

// New creates a new session directory.
func New(root, id, providerID, goal string) (*Session, error) {
	if id == "" {
		id = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	dir := filepath.Join(config.SessionsDir(root), id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Session{
		Dir: dir,
		State: State{
			ID:        id,
			CreatedAt: time.Now().UTC(),
			Provider:  providerID,
			Goal:      goal,
		},
		Events: events.NewWriter(filepath.Join(dir, "events.jsonl")),
	}
	if err := s.Save(); err != nil {
		return nil, err
	}
	return s, nil
}

// Label returns a human-readable session label (name + id, or id only).
func (s *Session) Label() string {
	if s == nil {
		return ""
	}
	name := strings.TrimSpace(s.State.Name)
	if name == "" {
		return s.State.ID
	}
	return name + " (" + s.State.ID + ")"
}

// SetName updates the session display name. Manual names are not overwritten by auto-naming.
func (s *Session) SetName(name string, manual bool) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	s.State.Name = sanitizeName(name)
	if manual {
		s.State.NameManual = true
	}
	return s.Save()
}

// NeedsAutoName reports whether the session should receive an LLM-generated title.
func (s *Session) NeedsAutoName() bool {
	if s == nil {
		return false
	}
	return strings.TrimSpace(s.State.Name) == "" && !s.State.NameManual
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "\"'`")
	if len(name) > 120 {
		name = strings.TrimSpace(name[:120])
	}
	return name
}

// Save writes state.json.
func (s *Session) Save() error {
	raw, err := json.MarshalIndent(s.State, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Dir, "state.json"), raw, 0o644)
}

// AddUsage merges provider usage into session stats.
func (s *Session) AddUsage(u provider.Usage, step int) {
	s.State.Stats.Requests++
	s.State.Stats.InputTokens += u.InputTokens
	s.State.Stats.OutputTokens += u.OutputTokens
	if u.InputTokens > 0 {
		s.State.Stats.LastInputTokens = u.InputTokens
	}
	s.State.Stats.TotalLatencyMs += u.LatencyMs
	s.State.Stats.EstimatedCostUSD += u.EstimatedCostUSD
	if step > s.State.Stats.Steps {
		s.State.Stats.Steps = step
	}
	s.State.Stats.UpdatedAt = time.Now().UTC()
}

// Load opens an existing session.
func Load(root, id string) (*Session, error) {
	dir := filepath.Join(config.SessionsDir(root), id)
	raw, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	return &Session{
		Dir:    dir,
		State:  state,
		Events: events.NewWriter(filepath.Join(dir, "events.jsonl")),
	}, nil
}
