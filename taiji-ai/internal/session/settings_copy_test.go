package session

import (
	"testing"
)

func TestCopySettingsDeepCopiesPointersAndSlices(t *testing.T) {
	temp := 0.7
	maxTok := 100
	src := Settings{
		Temperature:      &temp,
		MaxTokens:        &maxTok,
		SkillsAllow:      []string{"a"},
		FewShotExamples:  []ExampleTurn{{Role: "user", Content: "hi"}},
		ActorState:       map[string]string{"mood": "ok"},
		CouncilActors:    []string{"story"},
		FollowUpActor:    "story",
		BuiltinToolsDeny: []string{"shell"},
	}
	dst := CopySettings(src)

	src.SkillsAllow[0] = "mutated"
	src.FewShotExamples[0].Content = "changed"
	src.ActorState["mood"] = "bad"
	*src.Temperature = 0.1

	if dst.SkillsAllow[0] != "a" {
		t.Fatalf("skills allow=%v", dst.SkillsAllow)
	}
	if dst.FewShotExamples[0].Content != "hi" {
		t.Fatalf("few shot=%q", dst.FewShotExamples[0].Content)
	}
	if dst.ActorState["mood"] != "ok" {
		t.Fatalf("actor state=%v", dst.ActorState)
	}
	if *dst.Temperature != 0.7 {
		t.Fatalf("temperature=%v", *dst.Temperature)
	}
}

func TestCopySettingsNilActorState(t *testing.T) {
	dst := CopySettings(Settings{})
	if dst.ActorState != nil {
		t.Fatalf("actor state=%v", dst.ActorState)
	}
}
