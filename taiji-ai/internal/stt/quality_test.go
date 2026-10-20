package stt

import "testing"

func TestParseQuality(t *testing.T) {
	if n, err := ParseQuality("fast"); err != nil || n != 1 {
		t.Fatalf("fast: %v %d", err, n)
	}
	if n, err := ParseQuality(""); err != nil || n != 0 {
		t.Fatalf("empty: %v %d", err, n)
	}
	if _, err := ParseQuality("nope"); err == nil {
		t.Fatal("expected error")
	}
}
