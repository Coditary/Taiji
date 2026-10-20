package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/provider"
)

const compactMarker = "<!-- taiji:compact -->"

const compactSummaryPrompt = `Summarize the conversation below for the model's context window.
Preserve decisions, open tasks, file paths, errors, and technical details the assistant will need later.
Use the same language as the conversation when possible.
Reply with the summary only — no preamble.

Conversation:
`

// DefaultKeepRecentTurns is how many latest user/assistant turns stay verbatim after compaction.
const DefaultKeepRecentTurns = 2

// CompactOptions configures session compaction.
type CompactOptions struct {
	KeepRecentTurns int
}

// DefaultCompactOptions returns standard compaction settings.
func DefaultCompactOptions() CompactOptions {
	return CompactOptions{KeepRecentTurns: DefaultKeepRecentTurns}
}

// FormatCompactUserMessage stores a compact summary in session history.
func FormatCompactUserMessage(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return compactMarker
	}
	return compactMarker + "\n" + summary
}

// IsCompactUserMessage reports persisted compaction markers.
func IsCompactUserMessage(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), compactMarker)
}

// ParseCompactUserMessage returns the visible summary body.
func ParseCompactUserMessage(content string) (summary string, ok bool) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, compactMarker) {
		return content, false
	}
	rest := strings.TrimSpace(content[len(compactMarker):])
	return rest, true
}

// GenerateCompactSummary asks the provider to summarize a transcript.
func GenerateCompactSummary(ctx context.Context, providers *provider.Registry, providerID, transcript string) (string, error) {
	if providers == nil {
		return "", fmt.Errorf("provider registry is required")
	}
	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return "", fmt.Errorf("transcript is required")
	}

	p, err := providers.Get(strings.TrimSpace(providerID))
	if err != nil {
		p, err = providers.Default()
		if err != nil {
			return "", err
		}
	}

	resp, err := p.Complete(ctx, provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: "You write concise, accurate conversation summaries for coding agents."},
			{Role: provider.RoleUser, Content: compactSummaryPrompt + transcript},
		},
		MaxTokens:   2048,
		Temperature: 0.2,
	})
	if err != nil {
		return "", err
	}
	summary := strings.TrimSpace(resp.Content)
	if summary == "" {
		summary = strings.TrimSpace(provider.JoinThinkingBlocks(resp.ThinkingBlocks))
	}
	if summary == "" {
		return "", fmt.Errorf("empty compaction summary")
	}
	return summary, nil
}

// CompactSession replaces older turns with one compact summary user message plus a recent tail.
func CompactSession(ctx context.Context, sess *Session, providers *provider.Registry, providerID string, opts CompactOptions) (string, error) {
	if sess == nil {
		return "", fmt.Errorf("session is required")
	}
	keep := opts.KeepRecentTurns
	if keep < 0 {
		keep = DefaultKeepRecentTurns
	}

	turns := TurnsFromMessages(sess.State.Messages)
	if len(turns) == 0 {
		return "", fmt.Errorf("no conversation to compact")
	}

	split := len(turns) - keep
	if split <= 0 {
		return "", fmt.Errorf("not enough history to compact (need more than %d recent turns)", keep)
	}
	oldTurns := turns[:split]
	recentTurns := turns[split:]

	transcript := formatTurnsTranscript(oldTurns)
	summary, err := GenerateCompactSummary(ctx, providers, providerID, transcript)
	if err != nil {
		return "", err
	}

	var rebuilt []ChatTurn
	rebuilt = append(rebuilt, ChatTurn{User: summary, UserKind: "compact"})
	rebuilt = append(rebuilt, recentTurns...)

	sess.State.Messages = chatTurnsToMessages(rebuilt)
	if err := sess.Save(); err != nil {
		return "", err
	}
	return summary, nil
}

func formatTurnsTranscript(turns []ChatTurn) string {
	var b strings.Builder
	for _, t := range turns {
		if t.UserKind == "compact" {
			if body := strings.TrimSpace(t.User); body != "" {
				b.WriteString("[Prior summary]\n")
				b.WriteString(body)
				b.WriteString("\n\n")
			}
			continue
		}
		if u := strings.TrimSpace(t.User); u != "" {
			b.WriteString("User: ")
			b.WriteString(u)
			b.WriteString("\n")
		}
		if a := strings.TrimSpace(t.Assistant); a != "" {
			b.WriteString("Assistant: ")
			b.WriteString(a)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func chatTurnsToMessages(turns []ChatTurn) []provider.Message {
	var out []provider.Message
	for _, t := range turns {
		if t.UserKind == "compact" {
			out = append(out, provider.Message{
				Role:    provider.RoleUser,
				Content: FormatCompactUserMessage(t.User),
			})
			continue
		}
		user := strings.TrimSpace(t.User)
		asst := strings.TrimSpace(t.Assistant)
		if user != "" {
			out = append(out, provider.Message{Role: provider.RoleUser, Content: user})
		}
		if asst != "" {
			out = append(out, provider.Message{Role: provider.RoleAssistant, Content: asst})
		}
	}
	return out
}
