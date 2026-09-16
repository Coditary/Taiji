package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/config"
)

// Summary is a lightweight view of a stored session.
type Summary struct {
	ID        string
	Name      string
	Goal      string
	Provider  string
	CreatedAt time.Time
	Requests  int
	Messages  int
}

// ListAll returns session summaries under root, newest first.
func ListAll(root string) ([]Summary, error) {
	dir := config.SessionsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []Summary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summary, err := loadSummary(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		out = append(out, summary)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func loadSummary(dir string) (Summary, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		return Summary{}, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return Summary{}, fmt.Errorf("parse session state: %w", err)
	}
	return Summary{
		ID:        state.ID,
		Name:      strings.TrimSpace(state.Name),
		Goal:      state.Goal,
		Provider:  state.Provider,
		CreatedAt: state.CreatedAt,
		Requests:  state.Stats.Requests,
		Messages:  len(state.Messages),
	}, nil
}
