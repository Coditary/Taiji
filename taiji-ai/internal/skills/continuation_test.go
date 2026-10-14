package skills

import "testing"

func TestIsContinuationUserMessage(t *testing.T) {
	if !IsContinuationUserMessage(ContinuationPrompt) {
		t.Fatal("expected system continuation prompt to match legacy user filter")
	}
	if !IsContinuationUserMessage(legacyContinuationUserPrefix) {
		t.Fatal("expected legacy bare prefix to match")
	}
	if !IsContinuationUserMessage(legacyContinuationUserPrefix + ". Use them internally.\n\n## Skill: hallo") {
		t.Fatal("expected legacy continuation with skill body to match")
	}
	if IsContinuationUserMessage("Hallo") {
		t.Fatal("expected normal user text not to match")
	}
}
