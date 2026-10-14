// Memory eval runs repeated live agent trials for the Makefile/memory use case.
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
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/runtime"
)

const (
	successToken = "MEMORY_HIT_EXIT_42"
	targetScore  = 90.0
)

func main() {
	trials := envInt("MEMORY_EVAL_TRIALS", 10)
	minScore := envFloat("MEMORY_EVAL_MIN", targetScore)
	providerID := envStr("MEMORY_EVAL_PROVIDER", "gemma4")
	timeoutSec := envInt("MEMORY_EVAL_TIMEOUT", 180)

	root := strings.TrimSpace(os.Getenv("TAIJI_EVAL_ROOT"))
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cwd: %v\n", err)
			os.Exit(1)
		}
		root = filepath.Clean(wd)
	}

	memDir := filepath.Join(root, ".taiji", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "memory dir: %v\n", err)
		os.Exit(1)
	}
	if err := writeFixtureEntry(memDir); err != nil {
		fmt.Fprintf(os.Stderr, "fixture: %v\n", err)
		os.Exit(1)
	}

	cfg, err := taijicfg.Load(taijicfg.LoadOptions{Local: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	cfg.Root = root
	cfg.MCP.Servers = nil
	cfg.Memory = taijicfg.DefaultMemoryConfig()

	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Local: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "runtime: %v\n", err)
		os.Exit(1)
	}
	defer rt.Close()

	passed := 0
	var scores []float64
	for i := 1; i <= trials; i++ {
		var score float64
		var detail string
		var runErr error
		for attempt := 1; attempt <= 3; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
			score, detail, runErr = runTrial(ctx, rt, cfg, providerID, root)
			cancel()
			if runErr == nil || !isEmptyModelError(runErr) {
				break
			}
			if attempt < 3 {
				time.Sleep(3 * time.Second)
			}
		}
		if i < trials {
			time.Sleep(8 * time.Second)
		}
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "trial %d failed: %v (%s)\n", i, runErr, detail)
			scores = append(scores, 0)
			continue
		}
		scores = append(scores, score)
		if score >= minScore {
			passed++
		}
		fmt.Fprintf(os.Stderr, "trial %d: %.0f%% — %s\n", i, score, detail)
	}

	avg := average(scores)
	fmt.Fprintf(os.Stderr, "\nmemory eval: avg=%.1f%% pass=%d/%d (min=%.0f%%)\n", avg, passed, trials, minScore)
	if avg < minScore {
		os.Exit(1)
	}
}

func runTrial(ctx context.Context, rt *runtime.Runtime, cfg *taijicfg.Config, providerID, root string) (float64, string, error) {
	if rt == nil {
		return 0, "", fmt.Errorf("runtime required")
	}

	goal := "Run shell with command `exit 42`. If memory shows a verified fix, follow it and reply exactly " + successToken + "."

	var analysis trialAnalysis
	result, err := agent.Run(ctx, agent.RunRequest{
		Goal:       goal,
		ProviderID: providerID,
		Providers:  rt.Providers,
		Tools:      rt.Tools,
		Config:     cfg,
		RAG:        rt.RAG,
		Memory:     rt.Memory,
		WorkDir:    root,
		EventHandler: func(ev events.Event) {
			switch ev.Type {
			case events.TypeMemoryStored, events.TypeMemoryRevised:
				analysis.storedMemory = true
			case events.TypeMemoryRecalled:
				analysis.recallOK = true
			}
		},
	})
	if err != nil {
		return 0, "agent error", err
	}
	analyzeMessages(result.Messages, &analysis)
	analysis.answer = result.Answer

	score, detail := scoreTrial(analysis)
	return score, detail, nil
}

type trialAnalysis struct {
	toolError    bool
	recallOK     bool
	wrongKind    bool
	ranCheck     bool
	storedMemory bool
	answer       string
}

func analyzeMessages(msgs []provider.Message, a *trialAnalysis) {
	for _, m := range msgs {
		switch m.Role {
		case provider.RoleAssistant:
			for _, tc := range m.ToolCalls {
				trackToolCall(tc.Name, tc.Args, a)
			}
		case provider.RoleTool:
			lower := strings.ToLower(m.Content)
			if strings.Contains(lower, "error:") || strings.Contains(lower, "(error:") {
				if !isMemoryToolName(m.Name) {
					a.toolError = true
				}
			}
			if strings.Contains(lower, "[memory]") {
				a.recallOK = true
			}
			if m.Name == "shell" && strings.Contains(lower, "build_ok") {
				a.ranCheck = true
			}
		}
	}
}

func trackToolCall(name, args string, a *trialAnalysis) {
	switch name {
	case "memory_remember", "memory_revise":
		a.storedMemory = true
	case "memory_recall":
		kind := strings.ToLower(parseArg(args, "kind"))
		if kind == "preference" {
			a.wrongKind = true
		}
		if kind == "error" || kind == "pattern" || kind == "anti_pattern" || kind == "" {
			a.recallOK = true
		}
	case "shell":
		cmd := strings.ToLower(parseArg(args, "command"))
		if strings.Contains(cmd, "make check") {
			a.ranCheck = true
		}
	}
}

func scoreTrial(a trialAnalysis) (float64, string) {
	var pts float64
	var parts []string
	if a.toolError {
		pts += 25
		parts = append(parts, "error")
	}
	if a.recallOK && !a.wrongKind {
		pts += 25
		parts = append(parts, "recall")
	} else if a.wrongKind {
		parts = append(parts, "wrong-kind")
	}
	if strings.Contains(strings.ToUpper(a.answer), successToken) {
		pts += 25
		parts = append(parts, "answer")
	}
	if a.storedMemory {
		pts += 25
		parts = append(parts, "stored")
	}
	if len(parts) == 0 {
		parts = append(parts, "no-signals")
	}
	return pts, strings.Join(parts, ", ")
}

func writeFixtureEntry(memDir string) error {
	entry := "---\nkind: error\ntitle: Shell exit status 42\nfingerprint: \"exit status <n> exit 42*\"\ntags: [shell, test]\nhits: 0\nmisses: 0\nlast_verified: 2026-09-24\nrecall_text: |\n  shell command exits with status 42\n---\n\n# Verified fix\n\nReply exactly: " + successToken + "\n\n# Revision log\n\n- 2026-09-24: Seeded for memory eval\n"
	return os.WriteFile(filepath.Join(memDir, "eval-shell-exit-42.md"), []byte(entry), 0o644)
}

func isEmptyModelError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "empty response from model")
}

func isMemoryToolName(name string) bool {
	switch name {
	case "memory_recall", "memory_remember", "memory_revise":
		return true
	default:
		return false
	}
}

func parseArg(argsJSON, key string) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}
	v, _ := args[key].(string)
	return v
}

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

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
