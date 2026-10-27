package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

// ConversationHit is one search match in a stored chat session.
type ConversationHit struct {
	SessionID   string
	SessionName string
	Role        string
	Snippet     string
}

// SearchConversations scans persisted sessions under project root for keyword matches.
// Every whitespace-separated keyword must appear in the message (case-insensitive).
func SearchConversations(projectRoot, query string, limit int) ([]ConversationHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	keywords := strings.Fields(strings.ToLower(query))
	dir := config.SessionsDir(projectRoot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []ConversationHit
	for _, entry := range entries {
		if !entry.IsDir() || len(out) >= limit {
			continue
		}
		sessDir := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(filepath.Join(sessDir, "state.json"))
		if err != nil {
			continue
		}
		var state State
		if err := json.Unmarshal(raw, &state); err != nil {
			continue
		}
		name := strings.TrimSpace(state.Name)
		if name == "" {
			name = state.ID
		}
		messages := SanitizeStoredMessages(state.Messages, state.Settings)
		for _, m := range messages {
			if m.Role != provider.RoleUser && m.Role != provider.RoleAssistant {
				continue
			}
			content := strings.TrimSpace(m.Content)
			if content == "" || !messageMatchesKeywords(content, keywords) {
				continue
			}
			out = append(out, ConversationHit{
				SessionID:   state.ID,
				SessionName: name,
				Role:        string(m.Role),
				Snippet:     snippetConversation(content, 240),
			})
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func messageMatchesKeywords(content string, keywords []string) bool {
	lower := strings.ToLower(content)
	for _, kw := range keywords {
		if kw == "" || !strings.Contains(lower, kw) {
			return false
		}
	}
	return true
}

func snippetConversation(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
