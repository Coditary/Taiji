package hooks

import "testing"

func TestEntryMatchesEventToolBeforePrefix(t *testing.T) {
	entry := Entry{Event: EventToolBefore}
	if !entryMatchesEvent(entry, EventToolBefore+".read_file", "read_file") {
		t.Fatal("expected tool.before.read_file to match tool.before entry")
	}
	if entryMatchesEvent(entry, EventPromptBefore, "") {
		t.Fatal("prompt should not match tool.before")
	}
}

func TestMatcherMatchesPipeList(t *testing.T) {
	if !matcherMatches("read_file|write_file", "write_file") {
		t.Fatal("expected write_file match")
	}
	if matcherMatches("read_file", "shell") {
		t.Fatal("shell should not match")
	}
}

func TestConditionsMatchPathGlob(t *testing.T) {
	conds := []Condition{{MatchesAnyPath: "*.go", Tool: "write_file"}}
	if !conditionsMatch(conds, "write_file", []string{"/proj/main.go"}) {
		t.Fatal("expected path match")
	}
	if conditionsMatch(conds, "write_file", []string{"/proj/readme.md"}) {
		t.Fatal("readme should not match *.go")
	}
}

func TestToolEventName(t *testing.T) {
	if got := toolEventName(EventToolAfter, "glob"); got != EventToolAfter+".glob" {
		t.Fatalf("got %q", got)
	}
	if toolEventName(EventToolAfter, "") != EventToolAfter {
		t.Fatal("empty tool should return base event")
	}
}
