package memory

import (
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestSuggestKindsForGoal(t *testing.T) {
	cases := []struct {
		goal  string
		want  string
		empty bool
	}{
		{"Was ist meine Lieblingsfarbe?", "preference", false},
		{"what is my favorite color", "preference", false},
		{"wie heiße ich?", "preference", false},
		{"erinnere dich", "preference", false},
		{"how do I run tests in this repo", "workflow", false},
		{"Fix the compile error", "", true},
		{"Hello", "", true},
	}
	for _, tc := range cases {
		kinds := SuggestKindsForGoal(tc.goal)
		if tc.empty {
			if len(kinds) != 0 {
				t.Fatalf("goal %q: expected no kinds, got %v", tc.goal, kinds)
			}
			continue
		}
		if len(kinds) == 0 {
			t.Fatalf("goal %q: expected kinds", tc.goal)
		}
		found := false
		for _, kind := range kinds {
			if string(kind) == tc.want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("goal %q: got %v want to include %s", tc.goal, kinds, tc.want)
		}
	}
}

func TestRoutingAppendixPreference(t *testing.T) {
	svc := NewService(&taijicfg.Config{Root: t.TempDir()}, nil)
	appendix := RoutingAppendix(svc, "Was ist meine Lieblingsfarbe?")
	if !strings.Contains(appendix, "preference") || !strings.Contains(appendix, "memory_recall") {
		t.Fatalf("unexpected appendix: %q", appendix)
	}
}

func TestRoutingAppendixNameQuestion(t *testing.T) {
	svc := NewService(&taijicfg.Config{Root: t.TempDir()}, nil)
	appendix := RoutingAppendix(svc, "wie heiße ich?")
	if !strings.Contains(appendix, "preference") || !strings.Contains(appendix, "Call memory_recall") {
		t.Fatalf("unexpected appendix: %q", appendix)
	}
}
