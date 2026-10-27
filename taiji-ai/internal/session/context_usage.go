package session

// UsageBreakdownTotal sums all category estimates.
func UsageBreakdownTotal(b ContextUsageBreakdown) int {
	return b.SystemTokens + b.ToolsTokens + b.MCPTokens + b.RulesTokens + b.SkillsTokens + b.ConversationTokens
}

// ContextUsageBreakdown estimates how context window tokens are distributed.
type ContextUsageBreakdown struct {
	SystemTokens       int
	ToolsTokens        int
	MCPTokens          int
	RulesTokens        int
	SkillsTokens       int
	ConversationTokens int
}

// ScaleUsageBreakdown scales category estimates to match a known total token count.
func ScaleUsageBreakdown(b ContextUsageBreakdown, total int) ContextUsageBreakdown {
	if total <= 0 {
		return b
	}
	sum := b.SystemTokens + b.ToolsTokens + b.MCPTokens + b.RulesTokens + b.SkillsTokens + b.ConversationTokens
	if sum <= 0 {
		b.ConversationTokens = total
		return b
	}
	if sum == total {
		return b
	}
	scale := float64(total) / float64(sum)
	out := ContextUsageBreakdown{
		SystemTokens:       int(float64(b.SystemTokens) * scale),
		ToolsTokens:        int(float64(b.ToolsTokens) * scale),
		MCPTokens:          int(float64(b.MCPTokens) * scale),
		RulesTokens:        int(float64(b.RulesTokens) * scale),
		SkillsTokens:       int(float64(b.SkillsTokens) * scale),
		ConversationTokens: int(float64(b.ConversationTokens) * scale),
	}
	// Fix rounding drift on the largest bucket.
	diff := total - (out.SystemTokens + out.ToolsTokens + out.MCPTokens + out.RulesTokens + out.SkillsTokens + out.ConversationTokens)
	out.ConversationTokens += diff
	if out.ConversationTokens < 0 {
		out.ConversationTokens = 0
	}
	return out
}
