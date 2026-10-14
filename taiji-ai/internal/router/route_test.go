package router

import "testing"

func TestRouteTurnStop(t *testing.T) {
	for _, msg := range []string{"stop", "cancel", "/stop", "STOP"} {
		plan := RouteTurn(msg)
		if plan.Action != ActionStop {
			t.Fatalf("msg=%q action=%s want stop", msg, plan.Action)
		}
	}
}

func TestRouteTurnHook(t *testing.T) {
	plan := RouteTurn("/hook run-tests")
	if plan.Action != ActionHook {
		t.Fatalf("action=%s want hook", plan.Action)
	}
}

func TestRouteTurnNaturalLanguageIsUnknown(t *testing.T) {
	cases := []string{
		"Du heißt jetzt Luma",
		"Hey, wie heißt du?",
		"List files in the current directory",
		"kannst du mir sagen, was in diesem ordner drin liegt",
		"Kannst du mir sagen, was hier alles in dem repo drin ist",
		"¿Qué hay en este repositorio?",
		"Erkläre mir Quantenphysik ohne Kontext",
	}
	for _, msg := range cases {
		plan := RouteTurn(msg)
		if plan.Action != ActionUnknown {
			t.Fatalf("msg=%q action=%s want unknown", msg, plan.Action)
		}
	}
}

func TestRouteTurnEmpty(t *testing.T) {
	plan := RouteTurn("   ")
	if plan.Action != ActionUnknown {
		t.Fatalf("action=%s", plan.Action)
	}
}
