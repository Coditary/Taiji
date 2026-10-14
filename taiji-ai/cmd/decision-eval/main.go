// Decision eval runs repeated live agent trials for decide-tool routing accuracy.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/agent"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/decision"
	"github.com/coditary/taiji/internal/decisions"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

const targetScore = 80.0

type evalCase struct {
	goal              string
	expectDecide      bool
	expectProfile     string
	expectAdHoc       bool
	wantQuestionIDs   []string
}

var profileEvalCases = []evalCase{
	{goal: "Soll ich ein MacBook für 2800€ kaufen? Budget 3000€, altes Laptop langsam.", expectDecide: true, expectProfile: "buy"},
	{goal: "Lohnt sich der Kauf einer RTX 5090 jetzt oder besser warten?", expectDecide: true, expectProfile: "buy"},
	{goal: "Ich überlege mir eine Uhr für 500€ — was meinst du?", expectDecide: true, expectProfile: "buy"},
	{goal: "Should I buy this standing desk for $400? Budget is tight.", expectDecide: true, expectProfile: "buy"},
	{goal: "Mein Laptop ist 6 Jahre alt und langsam. Neues Gerät kostet ~1500€ — lohnt sich das?", expectDecide: true, expectProfile: "buy"},
	{goal: "Kann ich mir die neue Konsole leisten? Gehalt 3200€ netto, Konsole 550€.", expectDecide: true, expectProfile: "buy"},
	{goal: "Neues Headset kostet 180€, meins ist kaputt. Was denkst du?", expectDecide: true, expectProfile: "buy"},
	{goal: "Würde es Sinn machen, jetzt auf iPhone 17 zu wechseln? Vertrag läuft noch 4 Monate.", expectDecide: true, expectProfile: "buy"},
	{goal: "Was ist ein Decision Tree?", expectDecide: false},
	{goal: "Was kostet ein MacBook ungefähr?", expectDecide: false},
	{goal: "Welche Datei enthält die Decision-Config?", expectDecide: false},
	{goal: "Erkläre mir wie Kaufentscheidungen funktionieren.", expectDecide: false},
	{goal: "List files in the current directory.", expectDecide: false},
	{goal: "What is 2+2?", expectDecide: false},
	{goal: "Hello!", expectDecide: false},
	{goal: "Read the Makefile.", expectDecide: false},
}

var adhocEvalCases = []evalCase{
	{goal: "Help! My payouts failed — which team should handle this?", expectDecide: true, expectAdHoc: true, wantQuestionIDs: []string{"route", "urgent"}},
	{goal: "Customer says login is broken and they're angry. Who should take this?", expectDecide: true, expectAdHoc: true, wantQuestionIDs: []string{"route"}},
	{goal: "Should we deploy this hotfix to production tonight or wait until morning?", expectDecide: true, expectAdHoc: true, wantQuestionIDs: []string{"timing", "risk"}},
	{goal: "Wir überlegen ob wir Kandidatin Anna einstellen sollen. Starke Skills, aber hohes Gehalt.", expectDecide: true, expectAdHoc: true, wantQuestionIDs: []string{"fit", "hire"}},
	{goal: "Was ist ein Decision Tree?", expectDecide: false},
	{goal: "List files in the current directory.", expectDecide: false},
}

type stubDecisionBackend struct{}

func (stubDecisionBackend) Decide(_ context.Context, req decision.Request) (*decision.Response, error) {
	answers := map[string]decision.Answer{}
	for _, q := range req.Questions {
		switch q.Type {
		case decision.QuestionChoice:
			choice := ""
			if len(q.Criteria) > 0 {
				choice = q.Criteria[0]
			}
			answers[q.ID] = decision.Answer{ID: q.ID, Type: q.Type, Choice: &decision.ChoiceAnswer{Choice: choice}}
		case decision.QuestionNoul:
			answers[q.ID] = decision.Answer{ID: q.ID, Type: q.Type, Noul: &decision.NoulAnswer{Value: 0.7}}
		case decision.QuestionScore:
			answers[q.ID] = decision.Answer{ID: q.ID, Type: q.Type, Score: &decision.ScoreAnswer{Score: 0.8}}
		}
	}
	return &decision.Response{
		Backend: "stub",
		Driver:  "laya",
		Answers: answers,
	}, nil
}

func main() {
	rounds := envInt("DECISION_EVAL_ROUNDS", 1)
	minScore := envFloat("DECISION_EVAL_MIN", targetScore)
	providerID := envStr("DECISION_EVAL_PROVIDER", "gemma4")
	timeoutSec := envInt("DECISION_EVAL_TIMEOUT", 120)

	root := strings.TrimSpace(os.Getenv("TAIJI_EVAL_ROOT"))
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cwd: %v\n", err)
			os.Exit(1)
		}
		root = filepath.Clean(wd)
	}

	cfg, err := taijicfg.Load(taijicfg.LoadOptions{Local: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	mode := strings.ToLower(envStr("DECISION_EVAL_MODE", "profiles"))
	evalCases := profileEvalCases
	emptyProfilesDir := ""
	if mode == "adhoc" {
		evalCases = adhocEvalCases
		emptyProfilesDir = filepath.Join(root, ".taiji", "decisions-eval-empty")
		if err := os.MkdirAll(emptyProfilesDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "empty profiles dir: %v\n", err)
			os.Exit(1)
		}
		cfg.Decisions.Dirs = []string{emptyProfilesDir}
	}

	cfg.Root = root
	cfg.MCP.Servers = nil
	cfg.Memory.Enabled = boolPtr(false)

	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Local: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "runtime: %v\n", err)
		os.Exit(1)
	}
	defer rt.Close()

	toolReg := tools.NewRegistry()
	builtin.Register(toolReg)
	profileDirs := cfg.ResolvedDecision().Dirs
	if emptyProfilesDir != "" {
		profileDirs = []string{emptyProfilesDir}
	}
	profileLoader := decisions.NewLoader(profileDirs)
	saveDir := filepath.Join(root, ".taiji", "decisions")
	stubSvc := &decision.Service{Backend: stubDecisionBackend{}, DriverID: "laya", BackendName: "stub"}
	builtin.EnableDecisionOrchestration(toolReg, stubSvc, profileLoader, saveDir)
	fmt.Fprintf(os.Stderr, "decision eval mode: %s (%d cases)\n", mode, len(evalCases))

	var totalScore float64
	var totalCases int
	failures := 0

	for round := 1; round <= rounds; round++ {
		if rounds > 1 {
			fmt.Fprintf(os.Stderr, "\n=== round %d/%d ===\n", round, rounds)
		}
		for i, tc := range evalCases {
			var score float64
			var detail string
			var runErr error
			for attempt := 1; attempt <= 3; attempt++ {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
				score, detail, runErr = runCase(ctx, rt, cfg, toolReg, profileLoader, providerID, root, tc)
				cancel()
				if runErr == nil || !isEmptyModelError(runErr) {
					break
				}
				if attempt < 3 {
					time.Sleep(3 * time.Second)
				}
			}
			totalCases++
			totalScore += score
			status := "PASS"
			if score < 100 {
				status = "FAIL"
				failures++
			}
			fmt.Fprintf(os.Stderr, "[%s] case %d (%.0f%%): %s — %s\n", status, i+1, score, truncate(tc.goal, 60), detail)
			if runErr != nil {
				fmt.Fprintf(os.Stderr, "  error: %v\n", runErr)
			}
			time.Sleep(2 * time.Second)
		}
	}

	avg := totalScore / float64(totalCases)
	fmt.Fprintf(os.Stderr, "\ndecision eval: avg=%.1f%% failures=%d/%d (min=%.0f%%)\n", avg, failures, totalCases, minScore)
	if avg < minScore {
		os.Exit(1)
	}
}

func runCase(
	ctx context.Context,
	rt *runtime.Runtime,
	cfg *taijicfg.Config,
	toolReg *tools.Registry,
	profileLoader *decisions.Loader,
	providerID, root string,
	tc evalCase,
) (float64, string, error) {
	result, err := agent.Run(ctx, agent.RunRequest{
		Goal:             tc.goal,
		ProviderID:       providerID,
		Providers:        rt.Providers,
		Tools:            toolReg,
		Config:           cfg,
		DecisionProfiles: profileLoader,
		WorkDir:          root,
	})
	if err != nil {
		return 0, "agent error", err
	}

	called, profile, questionIDs := analyzeDecideCall(result.Messages)
	score, detail := scoreCase(tc, called, profile, questionIDs)
	return score, detail, nil
}

func analyzeDecideCall(msgs []provider.Message) (called bool, profile string, questionIDs []string) {
	for _, m := range msgs {
		if m.Role != provider.RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			if tc.Name != "decide" {
				continue
			}
			called = true
			profile = parseProfileArg(tc.Args)
			questionIDs = parseQuestionIDs(tc.Args)
		}
	}
	return called, profile, questionIDs
}

func parseQuestionIDs(argsJSON string) []string {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil
	}
	raw, ok := args["questions"].(map[string]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for id := range raw {
		out = append(out, strings.ToLower(strings.TrimSpace(id)))
	}
	return out
}

func parseProfileArg(argsJSON string) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}
	for _, key := range []string{"profile", "name", "decision"} {
		if v, ok := args[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.ToLower(strings.TrimSpace(v))
		}
	}
	return ""
}

func scoreCase(tc evalCase, called bool, profile string, questionIDs []string) (float64, string) {
	if tc.expectDecide {
		if !called {
			return 0, "expected decide, got none"
		}
		if tc.expectAdHoc {
			if profile != "" {
				return 40, "expected ad-hoc decide, got profile " + profile
			}
			if len(questionIDs) == 0 {
				return 50, "decide without custom questions"
			}
			if len(tc.wantQuestionIDs) > 0 {
				matched := countQuestionOverlap(questionIDs, tc.wantQuestionIDs)
				if matched == 0 {
					return 70, "adhoc decide, questions=" + strings.Join(questionIDs, ",")
				}
				if matched < len(tc.wantQuestionIDs) {
					return 85, "adhoc partial match " + strings.Join(questionIDs, ",")
				}
				return 100, "adhoc " + strings.Join(questionIDs, ",")
			}
			return 100, "adhoc " + strings.Join(questionIDs, ",")
		}
		if tc.expectProfile != "" {
			want := strings.ToLower(tc.expectProfile)
			if profile == want {
				return 100, "decide+" + profile
			}
			if profile != "" {
				return 50, "decide wrong profile: " + profile
			}
			if len(questionIDs) > 0 {
				return 70, "decide ad-hoc instead of profile " + want
			}
			return 70, "decide without profile (ad-hoc)"
		}
		return 100, "decide"
	}
	if called {
		return 0, "unexpected decide" + profileSuffix(profile)
	}
	return 100, "no decide"
}

func countQuestionOverlap(got, want []string) int {
	seen := map[string]struct{}{}
	for _, id := range got {
		seen[id] = struct{}{}
	}
	n := 0
	for _, id := range want {
		if _, ok := seen[strings.ToLower(id)]; ok {
			n++
		}
	}
	return n
}

func profileSuffix(profile string) string {
	if profile == "" {
		return ""
	}
	return " (" + profile + ")"
}

func isEmptyModelError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "empty response from model")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func boolPtr(v bool) *bool { return &v }

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	if n <= 0 {
		return fallback
	}
	return n
}

func envFloat(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var n float64
	fmt.Sscanf(v, "%f", &n)
	if n <= 0 {
		return fallback
	}
	return n
}

func envStr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
