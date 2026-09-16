package agent

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestUserRequestsThinking(t *testing.T) {
	cases := []string{
		"write a poem but think about it first",
		"kannst du auch vorher nachdenken",
		"think about this first",
		"<|tinkgin|>",
	}
	for _, goal := range cases {
		if !userRequestsThinking(goal) {
			t.Fatalf("expected thinking request: %q", goal)
		}
	}
}

func TestShouldRunPlanPass(t *testing.T) {
	if shouldRunPlanPass("Hello", taijicfg.ChainOfThoughtPrompt) {
		t.Fatal("prompt mode should not run plan pass")
	}
	if !shouldRunPlanPass("think about it and write a poem", taijicfg.ChainOfThoughtFull) {
		t.Fatal("explicit thinking request should run plan pass in full mode")
	}
	if !shouldRunPlanPass("How would you implement feature X?", taijicfg.ChainOfThoughtFull) {
		t.Fatal("complex task should run plan pass in full mode")
	}
	if shouldRunPlanPass("Hello", taijicfg.ChainOfThoughtOff) {
		t.Fatal("off mode should never run")
	}
}

func TestHasUnclosedThinkingBlock(t *testing.T) {
	open := thinkingOpenTag
	close := thinkingCloseTag
	if !hasUnclosedThinkingBlock(open + "still thinking") {
		t.Fatal("expected unclosed block")
	}
	if hasUnclosedThinkingBlock(open + "done" + close + "answer") {
		t.Fatal("expected closed block")
	}
}

func TestShouldUseThoughtChain(t *testing.T) {
	if shouldUseThoughtChain("Hallo!", taijicfg.ChainOfThoughtPrompt) {
		t.Fatal("greeting should not chain")
	}
	if !shouldUseThoughtChain("Kannst du darüber nachdenken wie P=NP bewiesen wird", taijicfg.ChainOfThoughtPrompt) {
		t.Fatal("explicit thinking should chain")
	}
	if !shouldUseThoughtChain("Explain step by step how to debug this", taijicfg.ChainOfThoughtPrompt) {
		t.Fatal("complex explain should chain")
	}
}

func TestUsesTaggedCoT(t *testing.T) {
	if !usesTaggedCoT(taijicfg.ChainOfThoughtPrompt) {
		t.Fatal("prompt should use tagged cot")
	}
	if usesTaggedCoT(taijicfg.ChainOfThoughtOff) {
		t.Fatal("off should not use tagged cot")
	}
}
