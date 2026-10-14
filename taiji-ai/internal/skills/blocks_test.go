package skills

import (
	"strings"
	"testing"
)

func TestBlocksFromInvokedOutput(t *testing.T) {
	combined := strings.Join([]string{
		FormatInvoked(Skill{Name: "hallo", Body: "Say hi."}),
		FormatInvoked(Skill{Name: "review", Body: "Check tests."}),
	}, "\n\n")
	blocks := BlocksFromInvokedOutput(combined)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Name != "hallo" || blocks[1].Name != "review" {
		t.Fatalf("unexpected names: %#v", blocks)
	}
	if !strings.Contains(blocks[0].Text, "Say hi.") || !strings.Contains(blocks[1].Text, "Check tests.") {
		t.Fatalf("unexpected block text: %#v", blocks)
	}
}
