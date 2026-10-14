package decisions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileBuyProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "buy", "DECISION.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `---
name: buy
description: Purchase decision
questions:
  worth_it:
    type: noul
    instructions: "Is it worth buying?"
  timing:
    type: choice
    instructions: "When to buy?"
    criteria: [now, wait, skip]
---
Notes for the agent.
`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	profile, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "buy" {
		t.Fatalf("name = %q", profile.Name)
	}
	if len(profile.Questions) != 2 {
		t.Fatalf("questions = %d", len(profile.Questions))
	}
}

func TestFindNormalizesUnderscores(t *testing.T) {
	all := []Profile{{Name: "buy", Path: "/tmp/buy/DECISION.md"}}
	if _, ok := Find(all, "buy"); !ok {
		t.Fatal("expected buy")
	}
	if _, ok := Find(all, "BUY"); !ok {
		t.Fatal("expected BUY")
	}
}

func TestSystemPromptAppendixListsProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "buy", "DECISION.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`---
name: buy
description: Purchase decision
questions:
  ok:
    type: noul
    instructions: "ok?"
---
`), 0o644); err != nil {
		t.Fatal(err)
	}
	appendix, err := SystemPromptAppendix(NewLoader([]string{dir}))
	if err != nil {
		t.Fatal(err)
	}
	if appendix == "" || !strings.Contains(appendix, "buy") {
		t.Fatalf("appendix = %q", appendix)
	}
	if !strings.Contains(appendix, "Workflow") {
		t.Fatal("appendix should include orchestration rules")
	}
}
