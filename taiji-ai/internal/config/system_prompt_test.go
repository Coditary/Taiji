package config

import (
	"strings"
	"testing"
)

func TestDefaultSystemPrompt(t *testing.T) {
	prompt := DefaultSystemPrompt()
	if strings.Contains(prompt, "<"+"think"+">") {
		t.Fatal("default prompt should not mention thinking tags")
	}
	if strings.Contains(prompt, "<|continue|>") {
		t.Fatal("default prompt should not mention continue token")
	}
	if !strings.Contains(prompt, "emit a tool call immediately") {
		t.Fatal("default prompt should require tool calls for actionable requests")
	}
}

func TestDefaultTaggedCoTSystemPrompt(t *testing.T) {
	prompt := DefaultTaggedCoTSystemPrompt()
	if !strings.Contains(prompt, "<"+"think"+">") {
		t.Fatal("prompt should contain thinking open tag")
	}
	if !strings.Contains(prompt, "</"+"think"+">") {
		t.Fatal("prompt should contain thinking close tag")
	}
	if !strings.Contains(prompt, "<|continue|>") {
		t.Fatal("prompt should describe continue token between thoughts")
	}
	if !strings.Contains(prompt, "ONE THOUGHT PER MESSAGE") {
		t.Fatal("prompt should require one thought per message")
	}
	if !strings.Contains(prompt, "### headings") {
		t.Fatal("prompt should forbid outline-style thinking")
	}
	if !strings.Contains(prompt, "nachdenken") {
		t.Fatal("prompt should mention explicit thinking requests")
	}
	if !strings.Contains(prompt, "Turn 1") {
		t.Fatal("prompt should demonstrate multi-turn chain example")
	}
}
