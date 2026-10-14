package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/coditary/taiji/internal/events"
)

var (
	reCodeAcronym  = regexp.MustCompile(`\b[A-Z]{2,}[A-Z0-9]*\b`)
	reCodeBacktick = regexp.MustCompile("`([^`]+)`")
	reCodePath     = regexp.MustCompile(`\binternal/[a-z][a-z0-9_/]*`)
)

var skipCodeSearchTerms = map[string]bool{
	"AI": true, "API": true, "CLI": true, "CSS": true, "CSV": true, "DNS": true,
	"EOF": true, "GPU": true, "HTML": true, "HTTP": true, "HTTPS": true, "ID": true,
	"JSON": true, "JS": true, "LLM": true, "OK": true, "PR": true,
	"SQL": true, "SSH": true, "TLS": true, "TODO": true, "TS": true, "UI": true,
	"URI": true, "URL": true, "UTF": true, "UTC": true, "XML": true, "YAML": true,
}

// runCodePrefetch greps the workspace for technical terms extracted from the user
// message (acronyms, paths, backticks). Language-agnostic — works for DE/ES/EN etc.
func runCodePrefetch(
	ctx context.Context,
	req RunRequest,
	goal string,
	emit func(events.Event),
) string {
	if req.Tools == nil {
		return ""
	}
	terms := extractCodeSearchTerms(goal)
	if len(terms) == 0 {
		return ""
	}

	searchPath := "."
	if wd := strings.TrimSpace(req.WorkDir); wd != "" {
		searchPath = wd
	}

	var blocks []string
	for _, term := range terms {
		args, _ := json.Marshal(map[string]any{
			"pattern":    term,
			"path":       searchPath,
			"head_limit": 25,
		})
		emit(events.Event{
			Type:     events.TypeToolCall,
			Step:     1,
			ToolName: "grep",
			Args:     string(args),
			Text:     "prefetch",
		})

		startArgs := string(args)
		result, err := req.Tools.Run(ctx, "grep", startArgs)
		obs := ""
		if err != nil {
			obs = "error: " + err.Error()
		} else {
			obs = strings.TrimSpace(result.Text)
			if obs == "" {
				obs = "no matches"
			}
		}
		emit(events.Event{
			Type:       events.TypeObservation,
			Step:       1,
			ToolName:   "grep",
			Text:       obs,
			DurationMs: 0,
		})
		if err == nil && strings.TrimSpace(result.Text) != "" && !strings.HasPrefix(obs, "no matches") {
			blocks = append(blocks, fmt.Sprintf("[code prefetch: %s]\n%s", term, truncatePrefetchText(result.Text, 4000)))
		}
	}
	if len(blocks) == 0 {
		return ""
	}
	return strings.Join(blocks, "\n\n")
}

func extractCodeSearchTerms(goal string) []string {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(term string) {
		term = strings.TrimSpace(term)
		if term == "" || len(term) < 2 {
			return
		}
		key := strings.ToLower(term)
		if skipCodeSearchTerms[strings.ToUpper(term)] || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, term)
	}

	for _, m := range reCodeAcronym.FindAllString(goal, -1) {
		add(m)
	}
	for _, m := range reCodeBacktick.FindAllStringSubmatch(goal, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range reCodePath.FindAllString(goal, -1) {
		add(m)
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

func truncatePrefetchText(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || len(text) <= max {
		return text
	}
	return text[:max] + "\n… (truncated)"
}
