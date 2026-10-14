package agent

import "testing"

func TestStripLeadingToolDebris(t *testing.T) {
	if got := stripLeadingToolDebris(`"]}Die Skills sind bereit.`); got != "Die Skills sind bereit." {
		t.Fatalf("unexpected: %q", got)
	}
	if got := stripLeadingToolDebris(`"}]`); got != "" {
		t.Fatalf("unexpected: %q", got)
	}
}
