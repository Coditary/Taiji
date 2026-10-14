package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/router"
	"github.com/coditary/taiji/internal/skills"
)

const skillSearchToolName = "skill_search"

func runSkillPrefetch(
	ctx context.Context,
	req RunRequest,
	goal string,
	emit func(events.Event),
) []router.SkillMatch {
	if req.SkillsCatalog == nil || !req.SkillsCatalog.PrefetchEnabled() {
		return nil
	}
	matches, err := req.SkillsCatalog.Search(ctx, goal)
	if err != nil || len(matches) == 0 {
		return nil
	}
	out := make([]router.SkillMatch, 0, len(matches))
	for _, m := range matches {
		out = append(out, router.SkillMatch{
			Name:        m.Name,
			Description: m.Description,
			Score:       m.Score,
			MatchSource: m.MatchSource,
			Fit:         m.Fit,
		})
	}
	emitSkillSearch(goal, out, emit)
	return out
}

func emitSkillSearch(goal string, matches []router.SkillMatch, emit func(events.Event)) {
	if emit == nil || len(matches) == 0 {
		return
	}
	topSkill := strings.TrimSpace(matches[0].Name)
	args, _ := json.Marshal(map[string]any{
		"query": strings.TrimSpace(goal),
	})
	emit(events.Event{
		Type:      events.TypeToolCall,
		Step:      1,
		ToolName:  skillSearchToolName,
		Args:      string(args),
		Text:      "search",
		SkillName: topSkill,
	})
	skillMatches := make([]skills.Match, 0, len(matches))
	for _, m := range matches {
		skillMatches = append(skillMatches, skills.Match{
			Name:        m.Name,
			Description: m.Description,
			Score:       m.Score,
			MatchSource: m.MatchSource,
			Fit:         m.Fit,
		})
	}
	emit(events.Event{
		Type:       events.TypeObservation,
		Step:       1,
		ToolName:   skillSearchToolName,
		Text:       skills.FormatMatches(skillMatches),
		SkillName:  topSkill,
		DurationMs: 0,
	})
}

func skillPrefetchAppendix(matches []router.SkillMatch) string {
	if len(matches) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Skill search (this turn)\n")
	b.WriteString("Ranked skill matches for the user message:\n")
	for i, m := range matches {
		line := strings.TrimSpace(m.Name)
		if m.Fit != "" {
			line += " — fit: " + m.Fit
		}
		if m.Score > 0 {
			line += " (" + formatSkillScore(m.Score)
			if src := strings.TrimSpace(m.MatchSource); src != "" {
				line += ", " + src
			}
			line += ")"
		}
		if desc := strings.TrimSpace(m.Description); desc != "" {
			line += " — " + desc
		}
		b.WriteString(strings.TrimSpace(line))
		if i < len(matches)-1 {
			b.WriteByte('\n')
		}
	}
	b.WriteString("\nIf a match looks helpful, call invoke_skill for the best fit before acting. Do not invoke low-fit skills unless the user explicitly asks.")
	return strings.TrimSpace(b.String())
}

func formatSkillScore(score float32) string {
	return fmt.Sprintf("%.2f", score)
}
