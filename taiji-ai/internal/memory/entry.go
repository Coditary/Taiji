package memory

import (
	"fmt"
	"strings"
	"time"
)

// Entry is a loaded memory document.
type Entry struct {
	ID           string
	Kind         Kind
	Title        string
	Fingerprint  string
	Tags          []string
	Keywords      []string
	RecallQueries []string
	Hits          int
	Misses       int
	LastVerified string
	RecallText   string
	VerifiedFix  string
	RevisionLog  string
	Scope        string
	Path         string
	Score        float32
	MatchSource  string
}

type entryFrontmatter struct {
	Kind         Kind     `yaml:"kind"`
	Title        string   `yaml:"title"`
	Fingerprint  string   `yaml:"fingerprint"`
	Tags          []string `yaml:"tags"`
	Keywords      []string `yaml:"keywords,omitempty"`
	RecallQueries []string `yaml:"recall_queries,omitempty"`
	Hits          int      `yaml:"hits"`
	Misses       int      `yaml:"misses"`
	LastVerified string   `yaml:"last_verified"`
	RecallText   string   `yaml:"recall_text"`
}

// RecallRequest searches memory entries.
type RecallRequest struct {
	Query       string
	Kind        string
	Fingerprint string
	ToolName    string
	TopK        int
}

// RememberRequest creates a new memory entry.
type RememberRequest struct {
	Kind        string
	Title       string
	RecallText  string
	VerifiedFix string
	Fingerprint string
	Tags          []string
	Keywords      []string
	RecallQueries []string
	Scope         string
}

// ReviseRequest updates an existing memory entry.
type ReviseRequest struct {
	ID           string
	RecallText   string
	VerifiedFix  string
	RevisionNote string
	Outcome       string
	Keywords      []string
	RecallQueries []string
}

func (e Entry) formatRecallBlock() string {
	var b strings.Builder
	source := e.MatchSource
	if source == "" {
		source = "memory"
	}
	fmt.Fprintf(&b, "[%s] id=%s kind=%s scope=%s", source, e.ID, e.Kind, e.Scope)
	if e.Score > 0 {
		fmt.Fprintf(&b, " score=%.3f", e.Score)
	}
	b.WriteString("\n")
	if e.Title != "" {
		fmt.Fprintf(&b, "Title: %s\n", e.Title)
	}
	if e.Fingerprint != "" {
		fmt.Fprintf(&b, "Fingerprint: %s\n", e.Fingerprint)
	}
	if e.RecallText != "" {
		fmt.Fprintf(&b, "Recall: %s\n", e.RecallText)
	}
	if len(e.Keywords) > 0 {
		fmt.Fprintf(&b, "Keywords: %s\n", strings.Join(e.Keywords, ", "))
	}
	if len(e.RecallQueries) > 0 {
		fmt.Fprintf(&b, "Recall queries: %s\n", strings.Join(e.RecallQueries, "; "))
	}
	if e.VerifiedFix != "" {
		fmt.Fprintf(&b, "Verified fix:\n%s\n", strings.TrimSpace(e.VerifiedFix))
	}
	if e.RevisionLog != "" {
		fmt.Fprintf(&b, "Revision log:\n%s\n", strings.TrimSpace(e.RevisionLog))
	}
	fmt.Fprintf(&b, "Path: %s", e.Path)
	return strings.TrimSpace(b.String())
}

func todayISO() string {
	return time.Now().UTC().Format("2006-01-02")
}

func extractSection(body, heading string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	marker := "# " + heading
	idx := strings.Index(strings.ToLower(body), strings.ToLower(marker))
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(marker):]
	rest = strings.TrimLeft(rest, " \t\r\n")
	if nl := strings.Index(rest, "\n# "); nl >= 0 {
		rest = rest[:nl]
	}
	return strings.TrimSpace(rest)
}

func appendRevisionLog(existing, note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return strings.TrimSpace(existing)
	}
	line := "- " + todayISO() + ": " + note
	if existing = strings.TrimSpace(existing); existing == "" {
		return line
	}
	return existing + "\n" + line
}
