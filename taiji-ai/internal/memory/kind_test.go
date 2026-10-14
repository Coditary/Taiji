package memory

import "testing"

func TestParseKind(t *testing.T) {
	k, err := ParseKind("anti_pattern")
	if err != nil || k != KindAntiPattern {
		t.Fatalf("kind=%q err=%v", k, err)
	}
	if _, err := ParseKind("bogus"); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}
