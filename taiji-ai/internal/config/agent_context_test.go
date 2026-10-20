package config

import "testing"

func TestMaxContextCharsDefault(t *testing.T) {
	if (AgentConfig{}).MaxContextCharsOrDefault() != defaultMaxContextChars {
		t.Fatal("default max context chars")
	}
	off := 0
	if (AgentConfig{MaxContextChars: &off}).MaxContextCharsOrDefault() != 0 {
		t.Fatal("zero disables")
	}
}
