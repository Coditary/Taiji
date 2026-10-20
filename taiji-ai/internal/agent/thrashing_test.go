package agent

import (
	"strings"
	"testing"
)

func TestThrashingEditCircuitBreaker(t *testing.T) {
	s := newThrashingState()
	args := `{"path":"a.go"}`
	var breaker string
	for i := 0; i < 3; i++ {
		breaker = s.observeToolOutcome("edit_file", args, "[DIAGNOSTICS WARNING]\nvet failed", false, 3)
	}
	if breaker == "" || !strings.Contains(breaker, "RUNTIME CIRCUIT BREAKER") {
		t.Fatalf("expected breaker, got %q", breaker)
	}
	if !strings.Contains(breaker, "a.go") {
		t.Fatalf("expected path in breaker: %q", breaker)
	}
	s.observeToolOutcome("edit_file", args, "edited a.go", false, 3)
	breaker = s.observeToolOutcome("edit_file", args, "[DIAGNOSTICS WARNING]\n", false, 3)
	if breaker != "" {
		t.Fatalf("expected counter reset after clean edit, got %q", breaker)
	}
}

func TestThrashingRunTestsCircuitBreaker(t *testing.T) {
	s := newThrashingState()
	failOut := "❌ FAIL: 1 failed\nFAIL: TestFoo (pkg/a)"
	var breaker string
	for i := 0; i < 3; i++ {
		breaker = s.observeToolOutcome("run_tests", `{}`, failOut, true, 3)
	}
	if breaker == "" || !strings.Contains(breaker, "FAIL: TestFoo") {
		t.Fatalf("expected breaker with test sig, got %q", breaker)
	}
	pass := s.observeToolOutcome("run_tests", `{}`, "✅ PASS: 1 tests across 1 packages (0.1s)", false, 3)
	if pass != "" {
		t.Fatal("pass should not trip breaker")
	}
	breaker = s.observeToolOutcome("run_tests", `{}`, failOut, true, 3)
	if breaker != "" {
		t.Fatal("expected test failure counter reset after pass")
	}
}
