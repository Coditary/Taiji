package session

import (
	"strings"
	"testing"
)

func TestGoalImpliesSceneChange(t *testing.T) {
	if !GoalImpliesSceneChange("ich streife durch die Wälder") {
		t.Fatal("expected movement")
	}
	if GoalImpliesSceneChange("Wie heißt du?") {
		t.Fatal("expected no movement")
	}
}

func TestBuildTurnSceneReminderStoryMention(t *testing.T) {
	root := t.TempDir()
	rem := BuildTurnSceneReminder(root, "@story Es geht weiter", Settings{Presentation: "narrative"})
	if rem == "" || !strings.Contains(rem, "Spielleiter") {
		t.Fatalf("reminder = %q", rem)
	}
}
