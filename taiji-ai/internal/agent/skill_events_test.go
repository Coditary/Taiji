package agent

import (
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools"
)

func TestEmitSkillToolActivity_splitsBatch(t *testing.T) {
	combined := strings.Join([]string{
		skills.FormatInvoked(skills.Skill{Name: "hallo", Body: "Hi"}),
		skills.FormatInvoked(skills.Skill{Name: "review", Body: "Check"}),
	}, "\n\n")
	var emitted []events.Event
	emit := func(ev events.Event) { emitted = append(emitted, ev) }
	tc := provider.ToolCall{
		ID:   "call_1",
		Name: skills.InvokeSkillToolName,
		Args: `{"names":["hallo","review"]}`,
	}
	outcome := toolExecOutcome{
		result: tools.RunResult{
			Text: combined,
			Data: map[string]any{"skill": "hallo, review", "skills": []string{"hallo", "review"}},
		},
	}
	observations := emitSkillToolActivity(emit, 1, tc, outcome, "", false)
	if len(observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(observations))
	}
	if len(emitted) != 4 {
		t.Fatalf("expected 4 events (2 calls + 2 obs), got %d", len(emitted))
	}
	if emitted[0].SkillName != "hallo" || emitted[2].SkillName != "review" {
		t.Fatalf("unexpected skill event names: %#v", emitted)
	}
}
