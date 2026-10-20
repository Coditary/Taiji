package router

import "strings"

// SystemPromptAppendix documents turn routing for the agent.
func SystemPromptAppendix() string {
	return strings.TrimSpace(`## Turn routing

Taiji may inject a short routing note for this turn. Interpret the user message in whatever language they use and pick tools yourself.

| action | meaning |
|--------|---------|
| unknown | no fixed route — read intent, then call the right tools |
| hook | explicit hook command — follow hook output |
| stop | do not continue this turn |

Use memory_recall when the user asks about stored facts; memory_remember when they ask you to remember something. Use list_dir, read_file, edit_file, grep, shell, rag_search, decide, invoke_skill, and ask_user as appropriate. Do not guess when inspection would help.`)
}

// PlanAppendix returns turn-specific routing guidance.
func PlanAppendix(plan Plan) string {
	if strings.TrimSpace(string(plan.Action)) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Turn routing (this turn)\n")
	b.WriteString("Routed action: **" + string(plan.Action) + "**")
	if reason := strings.TrimSpace(plan.Reason); reason != "" {
		b.WriteString(" — " + reason)
	}
	b.WriteString(".\n")
	switch plan.Action {
	case ActionUnknown:
		b.WriteString("No fixed keyword route. Interpret the user message (any language). Code grep prefetch may already be attached — use it, then act.\n")
		if skill := strings.TrimSpace(plan.SkillName); skill != "" {
			b.WriteString("Turn-start skill search suggests `" + skill + "` — use invoke_skill if it fits.\n")
		} else if len(plan.SkillMatches) > 0 {
			b.WriteString("Turn-start skill search ranked candidates — prefer the highest-fit match when invoking a skill.\n")
		}
	case ActionHook:
		b.WriteString("Follow hook output for this turn.\n")
	case ActionStop:
		b.WriteString("Stop processing. Reply briefly and do not call tools.\n")
	}
	return strings.TrimSpace(b.String())
}
