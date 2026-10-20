package agent

import (
	"context"
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools/builtin"
)

const defaultKeepRecentMessages = 6

// CompactSessionHistory compacts older conversation turns while preserving todos and checkpoints.
func CompactSessionHistory(ctx context.Context, sess *session.Session, providers *provider.Registry, providerID string, cfg *taijicfg.Config) (string, error) {
	if sess == nil {
		return "", fmt.Errorf("session is required")
	}
	turns := session.TurnsFromMessages(sess.State.Messages)
	if len(turns) == 0 {
		return "", fmt.Errorf("no conversation to compact")
	}
	keepTurns := keepRecentTurnsFromMessages(defaultKeepRecentMessages)
	split := len(turns) - keepTurns
	if split <= 0 {
		return "", fmt.Errorf("not enough history to compact")
	}
	oldTurns := turns[:split]
	recentTurns := turns[split:]

	preserve := builtin.PreservationBlockForSession(sess.State.ID)
	transcript := session.FormatTurnsTranscript(oldTurns)
	if preserve != "" {
		transcript += "\n\n" + preserve
	}

	summary, err := session.GenerateCompactSummary(ctx, providers, providerID, transcript)
	if err != nil {
		summary = deterministicCompactSummary(oldTurns, preserve)
	}
	if strings.TrimSpace(summary) == "" {
		return "", fmt.Errorf("empty compaction summary")
	}
	if preserve != "" && !strings.Contains(summary, "todo_plan") {
		summary += "\n\n" + preserve
	}

	rebuilt := []session.ChatTurn{{User: summary, UserKind: "compact"}}
	rebuilt = append(rebuilt, recentTurns...)
	sess.State.Messages = session.ChatTurnsToMessages(rebuilt)
	if err := sess.Save(); err != nil {
		return "", err
	}
	return summary, nil
}

// HistoryCharCount estimates persisted history size for compaction triggers.
func HistoryCharCount(messages []provider.Message) int {
	n := 0
	for _, m := range messages {
		n += len(m.Content)
		for _, tc := range m.ToolCalls {
			n += len(tc.Args) + len(tc.Name)
		}
	}
	return n
}

// ShouldAutoCompactHistory reports whether char-based compaction should run.
func ShouldAutoCompactHistory(messages []provider.Message, cfg *taijicfg.Config) bool {
	if cfg == nil {
		return false
	}
	limit := cfg.Agent.MaxContextCharsOrDefault()
	if limit <= 0 {
		return false
	}
	return HistoryCharCount(messages) > limit
}

func keepRecentTurnsFromMessages(keepMessages int) int {
	if keepMessages <= 0 {
		keepMessages = defaultKeepRecentMessages
	}
	// Each turn is roughly one user + one assistant message.
	if keepMessages < 2 {
		return 1
	}
	return (keepMessages + 1) / 2
}

func deterministicCompactSummary(turns []session.ChatTurn, preserve string) string {
	var b strings.Builder
	b.WriteString("[Offline compaction summary]\n")
	for _, t := range turns {
		if t.UserKind == "compact" {
			if body := strings.TrimSpace(t.User); body != "" {
				b.WriteString("- Prior summary retained\n")
			}
			continue
		}
		if u := strings.TrimSpace(t.User); u != "" {
			b.WriteString("- User: ")
			b.WriteString(truncateLine(u, 200))
			b.WriteString("\n")
		}
		if a := strings.TrimSpace(t.Assistant); a != "" {
			b.WriteString("- Assistant: ")
			b.WriteString(truncateLine(a, 200))
			b.WriteString("\n")
		}
	}
	if preserve != "" {
		b.WriteString("\n")
		b.WriteString(preserve)
	}
	return strings.TrimSpace(b.String())
}

func truncateLine(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
