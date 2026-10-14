package memory

import "testing"

func TestFingerprintNormalizesPathsAndNumbers(t *testing.T) {
	in := "build failed: /home/user/proj/main.go:42 undefined foo exit status 1"
	got := Fingerprint(in)
	if got == in {
		t.Fatalf("expected normalization, got %q", got)
	}
	if !containsAll(got, "<path>", ":<n>", "undefined", "foo") {
		t.Fatalf("unexpected fingerprint: %q", got)
	}
}

func TestMatchQualityPrefersSpecificPatterns(t *testing.T) {
	errText := "command failed exit status 42"
	generic := MatchQuality("exit status:<n>*", errText)
	specific := MatchQuality("exit status <n> exit 42*", errText)
	if specific <= generic {
		t.Fatalf("expected specific (%v) > generic (%v)", specific, generic)
	}
}

func TestMatchesFingerprintPrefix(t *testing.T) {
	if !MatchesFingerprint("go:undefined:*", "go build failed undefined foo bar") {
		t.Fatal("expected prefix match")
	}
	if !MatchesFingerprint("npm:enoent:*", "npm ERR! enoent some-package") {
		t.Fatal("expected npm enoent match")
	}
	if MatchesFingerprint("go:undefined:*", "python:import error") {
		t.Fatal("expected no match")
	}
}

func TestSlugFromTitle(t *testing.T) {
	if got := SlugFromTitle("Go: undefined identifier"); got != "go-undefined-identifier" {
		t.Fatalf("got %q", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if p == "" {
			continue
		}
		found := false
		for i := 0; i <= len(s)-len(p); i++ {
			if s[i:i+len(p)] == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
