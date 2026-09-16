package agent

import "testing"

func TestNeedsDeepReasoning(t *testing.T) {
	trivial := []string{
		"Hallo",
		"Hi there",
		"Sag mir ein kurzes Gedicht auf ohne nachzudenken",
		"Erzähl mir einen Witz?",
		"Was ist die Hauptstadt von Frankreich?",
	}
	for _, goal := range trivial {
		if needsDeepReasoning(goal) {
			t.Fatalf("should be trivial: %q", goal)
		}
	}

	deep := []string{
		"Wie würdest du Feature X in diesem Repo implementieren?",
		"Plan die Umsetzung Schritt für Schritt",
		"Debug this error and explain why it happens",
	}
	for _, goal := range deep {
		if !needsDeepReasoning(goal) {
			t.Fatalf("should need deep reasoning: %q", goal)
		}
	}
}
