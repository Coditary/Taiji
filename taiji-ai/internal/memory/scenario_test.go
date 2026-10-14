package memory

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

type memoryScenario struct {
	name       string
	userMsg    string
	action     IntentAction
	kind       Kind
	storeValue string
	recallQ    string
}

func TestMemoryScenarioEval(t *testing.T) {
	catalog := LoadKindCatalog(nil)
	scenarios := []memoryScenario{
		{name: "persona store de", userMsg: "Du heißt jetzt Luma", action: IntentStore, kind: KindPersona, storeValue: "Luma"},
		{name: "persona store en", userMsg: "From now on you are called Nova", action: IntentStore, kind: KindPersona, storeValue: "Nova"},
		{name: "user name de", userMsg: "Ich heiße Leonard", action: IntentStore, kind: KindPreference, storeValue: "Leonard"},
		{name: "user name en", userMsg: "My name is Alice", action: IntentStore, kind: KindPreference, storeValue: "Alice"},
		{name: "favorite color de", userMsg: "Meine Lieblingsfarbe ist Blau", action: IntentStore, kind: KindPreference, storeValue: "Blau"},
		{name: "favorite color en", userMsg: "My favorite color is green", action: IntentStore, kind: KindPreference, storeValue: "green"},
		{name: "language pref", userMsg: "Ich spreche Deutsch", action: IntentStore, kind: KindPreference, storeValue: "Deutsch"},
		{name: "repo pattern", userMsg: "In diesem Repo nutzen wir Tabs", action: IntentStore, kind: KindPattern, storeValue: "Tabs"},
		{name: "persona recall", userMsg: "Wie heißt du?", action: IntentRecall, kind: KindPersona, recallQ: "assistant name persona identity"},
		{name: "user recall", userMsg: "Wie heiße ich?", action: IntentRecall, kind: KindPreference, recallQ: "user name identity vorname"},
		{name: "color recall de", userMsg: "Was ist meine Lieblingsfarbe?", action: IntentRecall, kind: KindPreference, recallQ: "favorite color lieblingsfarbe preference"},
		{name: "color recall en", userMsg: "What is my favorite color?", action: IntentRecall, kind: KindPreference, recallQ: "favorite color lieblingsfarbe preference"},
	}

	passed := 0
	for _, sc := range scenarios {
		if runScenarioIntent(t, catalog, sc) {
			passed++
		}
	}

	rate := float64(passed) / float64(len(scenarios)) * 100
	t.Logf("memory scenario intent detection: %.0f%% (%d/%d)", rate, passed, len(scenarios))
	if rate < 90 {
		t.Fatalf("intent detection below 90%%: %.0f%%", rate)
	}

	passed = 0
	for _, sc := range scenarios {
		if sc.action != IntentStore {
			continue
		}
		if runScenarioStoreRecall(t, catalog, sc) {
			passed++
		}
	}
	storeScenarios := 0
	for _, sc := range scenarios {
		if sc.action == IntentStore {
			storeScenarios++
		}
	}
	rate = float64(passed) / float64(storeScenarios) * 100
	t.Logf("memory store+recall roundtrip: %.0f%% (%d/%d)", rate, passed, storeScenarios)
	if rate < 90 {
		t.Fatalf("store+recall below 90%%: %.0f%%", rate)
	}
}

func runScenarioIntent(t *testing.T, catalog *KindCatalog, sc memoryScenario) bool {
	t.Helper()
	intent := AnalyzeUserMessage(catalog, sc.userMsg)
	if intent == nil {
		t.Errorf("%s: no intent for %q", sc.name, sc.userMsg)
		return false
	}
	if intent.Action != sc.action {
		t.Errorf("%s: action=%s want %s", sc.name, intent.Action, sc.action)
		return false
	}
	if sc.action == IntentStore {
		if intent.Store == nil {
			t.Errorf("%s: missing store payload", sc.name)
			return false
		}
		if Kind(intent.Store.Kind) != sc.kind {
			t.Errorf("%s: kind=%s want %s", sc.name, intent.Store.Kind, sc.kind)
			return false
		}
		haystack := strings.ToLower(intent.Store.VerifiedFix + " " + intent.Store.RecallText)
		if !strings.Contains(haystack, strings.ToLower(sc.storeValue)) {
			t.Errorf("%s: store value missing %q in %q", sc.name, sc.storeValue, haystack)
			return false
		}
		return true
	}
	if len(intent.Kinds) == 0 || intent.Kinds[0] != sc.kind {
		t.Errorf("%s: kinds=%v want %s", sc.name, intent.Kinds, sc.kind)
		return false
	}
	if sc.recallQ != "" && intent.RecallQuery != sc.recallQ {
		t.Errorf("%s: recall query=%q want %q", sc.name, intent.RecallQuery, sc.recallQ)
		return false
	}
	return true
}

func runScenarioStoreRecall(t *testing.T, catalog *KindCatalog, sc memoryScenario) bool {
	t.Helper()
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	svc := NewService(cfg, nil)
	ctx := context.Background()

	intent := AnalyzeUserMessage(catalog, sc.userMsg)
	if intent == nil || intent.Store == nil {
		t.Errorf("%s: no store intent", sc.name)
		return false
	}
	if _, err := svc.Remember(ctx, *intent.Store); err != nil {
		t.Errorf("%s: remember failed: %v", sc.name, err)
		return false
	}

	recallIntent := AnalyzeUserMessage(catalog, recallQuestionFor(sc))
	if recallIntent == nil {
		t.Errorf("%s: no recall intent", sc.name)
		return false
	}
	entries, err := svc.Recall(ctx, RecallRequest{
		Query: recallIntent.RecallQuery,
		Kind:  string(sc.kind),
		TopK:  5,
	})
	if err != nil {
		t.Errorf("%s: recall failed: %v", sc.name, err)
		return false
	}
	if len(entries) == 0 {
		t.Errorf("%s: no recall hits for %q", sc.name, recallIntent.RecallQuery)
		return false
	}
	haystack := strings.ToLower(entries[0].VerifiedFix + " " + entries[0].RecallText)
	if !strings.Contains(haystack, strings.ToLower(sc.storeValue)) {
		t.Errorf("%s: recall missing %q in %q", sc.name, sc.storeValue, haystack)
		return false
	}
	return true
}

func recallQuestionFor(sc memoryScenario) string {
	switch sc.kind {
	case KindPersona:
		return "Wie heißt du?"
	case KindPattern:
		return "Was nutzen wir in diesem Repo?"
	default:
		if strings.Contains(strings.ToLower(sc.storeValue), "blau") || strings.Contains(strings.ToLower(sc.storeValue), "green") {
			return "Was ist meine Lieblingsfarbe?"
		}
		if strings.Contains(strings.ToLower(sc.userMsg), "sprache") {
			return "Welche Sprache spreche ich?"
		}
		return "Wie heiße ich?"
	}
}
