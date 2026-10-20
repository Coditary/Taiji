package session

import "testing"

func TestScaleUsageBreakdownPreservesTotal(t *testing.T) {
	b := ContextUsageBreakdown{
		SystemTokens:       500,
		ToolsTokens:        6800,
		SkillsTokens:       3300,
		ConversationTokens: 15200,
	}
	out := ScaleUsageBreakdown(b, 25_800)
	sum := out.SystemTokens + out.ToolsTokens + out.SkillsTokens + out.ConversationTokens
	if sum != 25_800 {
		t.Fatalf("sum = %d, want 25800", sum)
	}
}
