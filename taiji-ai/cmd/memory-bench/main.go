// Memory bench runs an independent scenario (Python probe error) and compares
// cold runs (no memory entry) vs warm runs (seeded memory + auto-recall).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/agent"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/runtime"
)

const (
	probeShellCmd = `python3 -c 'raise RuntimeError("TAIJI_DYNAMIC_PROBE_X7")'`
	successToken  = "DYNAMIC_PROBE_X7_OK"
	entryName     = "dynamic-python-probe-x7.md"
)

func main() {
	trials := envInt("MEMORY_BENCH_TRIALS", 3)
	providerID := envStr("MEMORY_BENCH_PROVIDER", "gemma4")
	timeoutSec := envInt("MEMORY_BENCH_TIMEOUT", 180)
	root := benchRoot()

	memDir := filepath.Join(root, ".taiji", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "memory dir: %v\n", err)
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
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory-isolated")

	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Local: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "runtime: %v\n", err)
		os.Exit(1)
	}
	defer rt.Close()

	fmt.Fprintf(os.Stderr, "memory bench root=%s provider=%s trials/phase=%d\n", root, providerID, trials)
	fmt.Fprintf(os.Stderr, "scenario: Python RuntimeError probe (unrelated to exit 42)\n\n")

	cold := runPhase("cold (no memory entry)", trials, timeoutSec, func() error {
		err := os.Remove(filepath.Join(memDir, entryName))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}, rt, cfg, providerID, root)

	time.Sleep(5 * time.Second)

	warm := runPhase("warm (memory seeded)", trials, timeoutSec, func() error {
		return writeProbeEntry(memDir)
	}, rt, cfg, providerID, root)

	printSummary(cold, warm)
	if warm.successRate < 0.67 {
		os.Exit(1)
	}
}

type phaseStats struct {
	name         string
	runs         []runResult
	avgMs        float64
	avgSteps     float64
	successRate  float64
	recallRate   float64
	speedupRatio float64
}

type runResult struct {
	durationMs int64
	steps      int
	recalled   bool
	success    bool
	detail     string
}

func runPhase(name string, trials, timeoutSec int, setup func() error, rt *runtime.Runtime, cfg *taijicfg.Config, providerID, root string) phaseStats {
	if err := setup(); err != nil {
		fmt.Fprintf(os.Stderr, "%s setup: %v\n", name, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "=== %s ===\n", name)
	stats := phaseStats{name: name, runs: make([]runResult, 0, trials)}
	for i := 1; i <= trials; i++ {
		if i > 1 {
			time.Sleep(6 * time.Second)
		}
		var res runResult
		var runErr error
		for attempt := 1; attempt <= 3; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
			res, runErr = runBenchTrial(ctx, rt, cfg, providerID, root)
			cancel()
			if runErr == nil || !isEmptyModelError(runErr) {
				break
			}
			if attempt < 3 {
				time.Sleep(3 * time.Second)
			}
		}
		if runErr != nil {
			res = runResult{detail: runErr.Error()}
		}
		stats.runs = append(stats.runs, res)
		fmt.Fprintf(os.Stderr, "  trial %d: %dms steps=%d recall=%v ok=%v — %s\n",
			i, res.durationMs, res.steps, res.recalled, res.success, res.detail)
	}
	stats.finalize()
	fmt.Fprintf(os.Stderr, "  avg: %.0fms steps=%.1f recall=%.0f%% success=%.0f%%\n\n",
		stats.avgMs, stats.avgSteps, stats.recallRate*100, stats.successRate*100)
	return stats
}

func (s *phaseStats) finalize() {
	if len(s.runs) == 0 {
		return
	}
	var msSum float64
	var stepSum float64
	var ok, recall int
	for _, r := range s.runs {
		if r.durationMs > 0 {
			msSum += float64(r.durationMs)
		}
		if r.steps > 0 {
			stepSum += float64(r.steps)
		}
		if r.success {
			ok++
		}
		if r.recalled {
			recall++
		}
	}
	n := float64(len(s.runs))
	s.avgMs = msSum / n
	s.avgSteps = stepSum / n
	s.successRate = float64(ok) / n
	s.recallRate = float64(recall) / n
}

func runBenchTrial(ctx context.Context, rt *runtime.Runtime, cfg *taijicfg.Config, providerID, root string) (runResult, error) {
	goal := strings.Join([]string{
		"Run the shell command:",
		"`" + probeShellCmd + "`.",
		"If a [memory] hint appears after the error, follow the verified fix and reply exactly",
		successToken + ".",
		"Do not guess the token without memory evidence.",
	}, " ")

	var recalled bool
	start := time.Now()
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
			if ev.Type == events.TypeMemoryRecalled {
				recalled = true
			}
		},
	})
	elapsed := time.Since(start)
	if err != nil {
		return runResult{durationMs: elapsed.Milliseconds(), detail: err.Error()}, err
	}

	success := strings.Contains(strings.ToUpper(result.Answer), successToken)
	detail := "answer"
	if !success {
		detail = "missing token in answer"
	}
	if recalled {
		detail += ", auto-recall"
	}
	return runResult{
		durationMs: elapsed.Milliseconds(),
		steps:      result.Steps,
		recalled:   recalled,
		success:    success,
		detail:     detail,
	}, nil
}

func printSummary(cold, warm phaseStats) {
	if cold.avgMs > 0 && warm.avgMs > 0 {
		warm.speedupRatio = cold.avgMs / warm.avgMs
	}
	fmt.Fprintf(os.Stderr, "=== summary ===\n")
	fmt.Fprintf(os.Stderr, "cold:  avg=%.0fms steps=%.1f success=%.0f%%\n", cold.avgMs, cold.avgSteps, cold.successRate*100)
	fmt.Fprintf(os.Stderr, "warm:  avg=%.0fms steps=%.1f recall=%.0f%% success=%.0f%%\n",
		warm.avgMs, warm.avgSteps, warm.recallRate*100, warm.successRate*100)
	if warm.speedupRatio > 1 {
		fmt.Fprintf(os.Stderr, "speedup: %.1fx faster with memory (wall time)\n", warm.speedupRatio)
	} else if cold.avgMs > 0 && warm.avgMs > 0 {
		fmt.Fprintf(os.Stderr, "speedup: no wall-time gain (warm/cold = %.2f)\n", warm.avgMs/cold.avgMs)
	}
	if warm.recallRate >= 0.67 && warm.successRate >= 0.67 {
		fmt.Fprintf(os.Stderr, "dynamic recall: OK (independent Python probe, not exit-42 fixture)\n")
	}
}

func writeProbeEntry(memDir string) error {
	entry := "---\nkind: pattern\ntitle: Python dynamic probe X7\nfingerprint: \"runtimeerror taiji dynamic probe x7*\"\ntags: [python, bench, dynamic]\nhits: 0\nmisses: 0\nlast_verified: 2026-09-24\nrecall_text: |\n  Python RuntimeError TAIJI_DYNAMIC_PROBE_X7 during independent bench\n---\n\n# Verified fix\n\nReply exactly: " + successToken + "\n\n# Revision log\n\n- 2026-09-24: Independent dynamic bench entry\n"
	return os.WriteFile(filepath.Join(memDir, entryName), []byte(entry), 0o644)
}

func benchRoot() string {
	if v := strings.TrimSpace(os.Getenv("TAIJI_BENCH_ROOT")); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(os.TempDir(), "taiji-memory-bench-dynamic")
}

func isEmptyModelError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "empty response from model")
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

func envStr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
