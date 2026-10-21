package session

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
)

// TruncateBeforeTurn keeps complete turns [0, throughTurn] and drops later turns.
func TruncateBeforeTurn(messages []provider.Message, firstTurnToDrop int) []provider.Message {
	if firstTurnToDrop < 0 || len(messages) == 0 {
		return append([]provider.Message(nil), messages...)
	}

	turn := -1
	pendingUser := false
	pendingUserStart := -1

	for i, msg := range messages {
		switch msg.Role {
		case provider.RoleUser:
			content := strings.TrimSpace(msg.Content)
			if skills.IsContinuationUserMessage(content) || IsHiddenUserMessage(content) {
				continue
			}
			if IsCompactUserMessage(content) {
				if pendingUser {
					turn++
					if turn >= firstTurnToDrop {
						return messages[:pendingUserStart]
					}
					pendingUser = false
				}
				turn++
				if turn >= firstTurnToDrop {
					return messages[:i]
				}
				continue
			}
			if pendingUser && isRepeatUserPrompt(messages[pendingUserStart].Content, content) {
				continue
			}
			if pendingUser {
				turn++
				if turn >= firstTurnToDrop {
					return messages[:pendingUserStart]
				}
			}
			turn++
			pendingUserStart = i
			pendingUser = true
			if turn >= firstTurnToDrop {
				return messages[:i]
			}
		case provider.RoleAssistant:
			content := strings.TrimSpace(msg.Content)
			if !pendingUser || content == "" {
				continue
			}
			pendingUser = false
		}
	}
	return append([]provider.Message(nil), messages...)
}

// BranchFrom creates a new session copied from src through throughTurn (inclusive).
// throughTurn < 0 copies the full message history.
func BranchFrom(root string, src *Session, throughTurn int) (*Session, error) {
	if src == nil {
		return nil, fmt.Errorf("source session is nil")
	}
	messages := src.State.Messages
	if throughTurn >= 0 {
		messages = TruncateBeforeTurn(src.State.Messages, throughTurn+1)
	} else {
		messages = cloneMessages(src.State.Messages)
	}

	providerID := strings.TrimSpace(src.State.Provider)
	goal := src.State.Goal
	if goal == "" {
		goal = "tui-chat"
	}

	out, err := New(root, "", providerID, goal)
	if err != nil {
		return nil, err
	}

	out.State.Messages = cloneMessages(messages)
	out.State.Settings = src.State.Settings
	out.State.Stats = src.State.Stats
	out.State.Name = branchSessionName(src.State.Name)
	out.State.NameManual = true

	eventsPath := filepath.Join(src.Dir, "events.jsonl")
	if all, err := events.ReadFile(eventsPath); err == nil && len(all) > 0 {
		kept := EventsThroughTurn(all, throughTurn)
		for i := range kept {
			kept[i].SessionID = out.State.ID
		}
		if err := events.WriteFile(filepath.Join(out.Dir, "events.jsonl"), kept); err != nil {
			return nil, err
		}
	}

	if err := out.Save(); err != nil {
		return nil, err
	}
	return out, nil
}

func cloneMessages(in []provider.Message) []provider.Message {
	out := make([]provider.Message, len(in))
	for i, m := range in {
		out[i] = m
		if len(m.ToolCalls) > 0 {
			out[i].ToolCalls = append([]provider.ToolCall(nil), m.ToolCalls...)
		}
	}
	return out
}

func branchSessionName(existing string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return "Branch"
	}
	if strings.HasSuffix(existing, " (branch)") {
		return existing
	}
	return existing + " (branch)"
}
