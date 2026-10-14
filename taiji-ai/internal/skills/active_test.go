package skills

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestActiveFromMessagesBatchOutput(t *testing.T) {
	combined := FormatInvoked(Skill{Name: "hallo", Body: "Hi"}) + "\n\n" + FormatInvoked(Skill{Name: "review", Body: "Check"})
	msgs := []provider.Message{
		{Role: provider.RoleTool, Name: InvokeSkillToolName, Content: combined},
	}
	active := ActiveFromMessages(msgs)
	if len(active) != 2 || active[0] != "hallo" || active[1] != "review" {
		t.Fatalf("unexpected active skills: %#v", active)
	}
}

func TestActiveFromMessages(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleTool, Name: InvokeSkillToolName, Content: FormatInvoked(Skill{Name: "hallo", Body: "Hi"})},
		{Role: provider.RoleTool, Name: InvokeSkillToolName, Content: "error: unknown skill"},
		{Role: provider.RoleTool, Name: InvokeSkillToolName, Content: FormatInvoked(Skill{Name: "review", Body: "Check"})},
	}
	active := ActiveFromMessages(msgs)
	if len(active) != 2 || active[0] != "hallo" || active[1] != "review" {
		t.Fatalf("unexpected active skills: %#v", active)
	}
}
