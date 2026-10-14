package decisions

import (
	"fmt"
	"sort"
	"strings"
)

const (
	decideToolName            = "decide"
	decisionRememberToolName  = "decision_remember"
)

// SystemPromptAppendix lists saved decision profiles for the agent system prompt.
func SystemPromptAppendix(loader *Loader) (string, error) {
	if loader == nil {
		return OrchestrationAppendix(nil), nil
	}
	all, err := loader.LoadAll()
	if err != nil {
		return "", err
	}
	return BuildFullAppendix(Invokable(all)), nil
}

// Invokable returns profiles the agent may call via decide(profile=…).
func Invokable(all []Profile) []Profile {
	out := make([]Profile, 0, len(all))
	for _, p := range all {
		if p.DisableModelInvocation {
			continue
		}
		out = append(out, p)
	}
	return out
}

// OrchestrationAppendix returns static decide-tool guidance (no profile catalog).
func OrchestrationAppendix(profiles []Profile) string {
	var b strings.Builder
	b.WriteString("## Decisions\n\n")
	b.WriteString("Structured decision questions (choice, noul, score) run through `" + decideToolName + "` and the configured decision engine.\n\n")

	b.WriteString("### Workflow\n")
	b.WriteString("1. **Saved profile fits?** → `" + decideToolName + "({\"profile\": \"<name>\", \"text\": \"<situation>\"})`\n")
	b.WriteString("2. **No profile fits?** → design 1–4 custom questions, then `" + decideToolName + "({\"questions\": {...}, \"text\": \"<situation>\"})`\n")
	b.WriteString("3. **Schema reusable?** → add `\"save_as\": \"short-name\"` to the same custom `" + decideToolName + "` call, or call `" + decisionRememberToolName + "` afterward\n")
	b.WriteString("4. **Next time** → reuse the saved profile from step 3 instead of rebuilding questions\n\n")

	b.WriteString("### When to call `" + decideToolName + "`\n")
	b.WriteString("- The user needs a structured evaluation: buy/wait/skip, route to a team, hire/no-hire, deploy/staging, urgency, risk, or similar multi-factor choices.\n")
	b.WriteString("- You can decompose the situation into choice / noul / score questions.\n")
	b.WriteString("- Step 1: emit `" + decideToolName + "` immediately. Do not answer from general knowledge first.\n\n")

	b.WriteString("### Custom (on-demand) questions\n")
	b.WriteString("- Use when **no saved profile matches**, even if the topic is not in the catalog.\n")
	b.WriteString("- `choice` needs `criteria` (option labels). `noul` = yes/no likelihood 0–1. `score` rates against criteria.\n")
	b.WriteString("- Put the full situation into `text`; put the evaluation dimensions into `questions`.\n")
	b.WriteString("- Good names: `route`, `urgent`, `risk_level`, `timing`, `worth_it`.\n\n")

	b.WriteString("### When NOT to call `" + decideToolName + "`\n")
	b.WriteString("- Greetings, thanks, small talk, math, or general facts.\n")
	b.WriteString("- File/shell/search tasks (use filesystem tools instead).\n")
	b.WriteString("- User asks what decisions/decide **are** (explain conceptually).\n")
	b.WriteString("- User asks **you** to pick a file, command, or code change — use normal tools, not `" + decideToolName + "`.\n\n")

	b.WriteString("### Examples\n")
	b.WriteString("User: Help! My payouts failed — which team should handle this?\n")
	b.WriteString("Assistant: (structured tool call: " + decideToolName + " with custom questions route+urgent, text with the situation, optional save_as: \"support-route\")\n\n")
	b.WriteString("User: Soll ich ein MacBook für 2800€ kaufen? Budget 3000€.\n")
	b.WriteString("Assistant: (structured tool call: " + decideToolName + " with profile \"buy\" if listed, else custom purchase questions)\n\n")
	b.WriteString("User: Was ist ein Decision Tree?\n")
	b.WriteString("Assistant: (plain text explanation — no " + decideToolName + ")\n")

	if len(profiles) > 0 {
		b.WriteString("\n\n### Saved profiles\n\n")
		for _, p := range profiles {
			line := "- " + p.Name
			if desc := strings.TrimSpace(p.Description); desc != "" {
				line += ": " + desc
			}
			if when := strings.TrimSpace(p.When); when != "" {
				line += " (" + when + ")"
			}
			b.WriteString(line + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

// BuildFullAppendix combines orchestration rules with the profile catalog.
func BuildFullAppendix(profiles []Profile) string {
	if len(profiles) == 0 {
		return OrchestrationAppendix(nil)
	}
	sorted := append([]Profile(nil), profiles...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	return OrchestrationAppendix(sorted)
}

// BuildCatalogAppendix formats saved decisions for the system prompt.
func BuildCatalogAppendix(profiles []Profile) string {
	return BuildFullAppendix(profiles)
}

// FormatList returns a human-readable catalog for /decisions.
func FormatList(profiles []Profile) string {
	invokable := Invokable(profiles)
	if len(invokable) == 0 {
		return "Keine gespeicherten Decisions gefunden.\nLege Profile unter .taiji/decisions/<name>/DECISION.md ab oder speichere on-demand Schemas mit decision_remember."
	}
	sort.Slice(invokable, func(i, j int) bool { return invokable[i].Name < invokable[j].Name })
	var b strings.Builder
	b.WriteString("Verfügbare Decisions:\n")
	for _, p := range invokable {
		line := "- " + p.Name
		if desc := strings.TrimSpace(p.Description); desc != "" {
			line += ": " + desc
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(fmt.Sprintf("\nAufruf: decide({\"profile\": \"%s\", \"text\": \"…\"})", invokable[0].Name))
	return strings.TrimSpace(b.String())
}
