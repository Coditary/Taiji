package hooks

import (
	"encoding/json"
	"os"
	"strings"
)

// claudeHooksFile is the Claude Code-compatible hooks.json shape (subset).
type claudeHooksFile struct {
	Version int                       `json:"version"`
	Hooks   map[string][]claudeHookGroup `json:"hooks"`
}

type claudeHookGroup struct {
	Matcher string              `json:"matcher,omitempty"`
	Hooks   []claudeHookAction  `json:"hooks"`
}

type claudeHookAction struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// LoadClaudeJSON reads hooks.json and converts supported events to Entries.
func LoadClaudeJSON(path string) ([]Entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc claudeHooksFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var out []Entry
	for eventName, groups := range doc.Hooks {
		taijiEvent := claudeEventToTaiji(eventName)
		if taijiEvent == "" {
			continue
		}
		for i, group := range groups {
			entry := Entry{
				ID:      sanitizeID(eventName + "-" + itoa(i)),
				Event:   taijiEvent,
				Matcher: group.Matcher,
			}
			for _, hook := range group.Hooks {
				if strings.ToLower(strings.TrimSpace(hook.Type)) != "command" {
					continue
				}
				cmd := strings.TrimSpace(hook.Command)
				if cmd == "" {
					continue
				}
				entry.Actions = append(entry.Actions, ActionSpec{
					Command: cmd,
					Timeout: hook.Timeout,
				})
			}
			if len(entry.Actions) > 0 {
				out = append(out, entry)
			}
		}
	}
	return out, nil
}

func claudeEventToTaiji(name string) string {
	switch strings.TrimSpace(name) {
	case "PreToolUse":
		return EventToolBefore
	case "PostToolUse", "PostToolUseFailure":
		return EventToolAfter
	case "UserPromptSubmit":
		return EventPromptBefore
	case "Stop":
		return EventSessionIdle
	case "SessionStart":
		return EventSessionStart
	case "SessionEnd":
		return EventSessionError
	default:
		return ""
	}
}

func sanitizeID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "hook"
	}
	var b strings.Builder
	for _, ch := range s {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9', ch == '-', ch == '_':
			b.WriteRune(ch)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}
