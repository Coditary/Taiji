package agent

import (
	"strings"
	"time"

	"github.com/coditary/taiji/internal/provider"
)

// RunMetrics summarizes timing and throughput for one user request.
type RunMetrics struct {
	TotalDurationMs      int64
	WaitDurationMs       int64
	ThinkingDurationMs   int64
	GenerationDurationMs int64
	OutputTokens         int
	InputTokens          int
	EstimatedCostUSD     float64
	TokensPerSec         float64
	HadThinking          bool
	ToolCalls            int
	ToolDurationMs       int64
	Steps                int
	ContextTokens        int
	ContextWindow        int
	ContextUsedPct       float64
}

type runMetricsCollector struct {
	start time.Time

	waitDuration time.Duration
	waitStart    time.Time
	inWait       bool

	thinkingDuration time.Duration
	thinkingStart    time.Time
	inThinking       bool

	genStart        time.Time
	genEnd          time.Time
	inGeneration    bool
	outputTokens    int
	inputTokens     int
	estimatedCost   float64
	lastProviderTPS float64

	toolCalls      int
	toolDurationMs int64

	finished bool
}

func newRunMetricsCollector() *runMetricsCollector {
	return &runMetricsCollector{start: time.Now()}
}

func (c *runMetricsCollector) OnCompletionStart() {
	if c == nil || c.finished {
		return
	}
	if c.inWait {
		return
	}
	c.inWait = true
	c.waitStart = time.Now()
}

func (c *runMetricsCollector) OnCompletionFinished() {
	if c == nil || c.finished {
		return
	}
	c.pauseWait()
}

func (c *runMetricsCollector) pauseWait() {
	if c == nil || !c.inWait {
		return
	}
	c.waitDuration += time.Since(c.waitStart)
	c.inWait = false
}

func (c *runMetricsCollector) OnThinkingActivity() {
	if c == nil || c.finished {
		return
	}
	c.pauseWait()
	if !c.inThinking {
		c.inThinking = true
		c.thinkingStart = time.Now()
	}
}

func (c *runMetricsCollector) pauseThinking() {
	if c == nil || !c.inThinking {
		return
	}
	c.thinkingDuration += time.Since(c.thinkingStart)
	c.inThinking = false
}

func (c *runMetricsCollector) OnAnswerDelta() {
	if c == nil || c.finished {
		return
	}
	c.pauseWait()
	c.pauseThinking()
	now := time.Now()
	if !c.inGeneration {
		c.inGeneration = true
		c.genStart = now
	}
	c.genEnd = now
}

func (c *runMetricsCollector) AddUsage(u provider.Usage) {
	if c == nil {
		return
	}
	if u.InputTokens > 0 {
		c.inputTokens += u.InputTokens
	}
	if u.OutputTokens > 0 {
		c.outputTokens += u.OutputTokens
	}
	if u.EstimatedCostUSD > 0 {
		c.estimatedCost += u.EstimatedCostUSD
	}
	if u.TokensPerSec > 0 {
		c.lastProviderTPS = u.TokensPerSec
	}
}

func (c *runMetricsCollector) AddToolExecution(durationMs int64) {
	if c == nil || c.finished {
		return
	}
	c.toolCalls++
	if durationMs > 0 {
		c.toolDurationMs += durationMs
	}
}

func (c *runMetricsCollector) Finalize(finalAnswer string) RunMetrics {
	if c == nil {
		return RunMetrics{}
	}
	c.finished = true
	c.pauseWait()
	c.pauseThinking()

	total := time.Since(c.start)
	if c.outputTokens == 0 && strings.TrimSpace(finalAnswer) != "" {
		c.outputTokens = estimateTokens(finalAnswer)
	}

	genDur := time.Duration(0)
	if c.inGeneration {
		genDur = c.genEnd.Sub(c.genStart)
	}

	tps := 0.0
	if genDur >= 50*time.Millisecond && c.outputTokens > 0 {
		tps = float64(c.outputTokens) / genDur.Seconds()
	} else if c.lastProviderTPS > 0 {
		tps = c.lastProviderTPS
	} else if c.outputTokens > 0 {
		remaining := total - c.thinkingDuration
		if remaining > 0 {
			tps = float64(c.outputTokens) / remaining.Seconds()
		}
	}

	return RunMetrics{
		TotalDurationMs:      total.Milliseconds(),
		WaitDurationMs:       c.waitDuration.Milliseconds(),
		ThinkingDurationMs:   c.thinkingDuration.Milliseconds(),
		GenerationDurationMs: genDur.Milliseconds(),
		OutputTokens:         c.outputTokens,
		InputTokens:          c.inputTokens,
		EstimatedCostUSD:     c.estimatedCost,
		TokensPerSec:         tps,
		HadThinking:          c.thinkingDuration > 0,
		ToolCalls:            c.toolCalls,
		ToolDurationMs:       c.toolDurationMs,
	}
}

func turnMetricsEventData(m RunMetrics) map[string]any {
	return map[string]any{
		"total_duration_ms":      m.TotalDurationMs,
		"wait_duration_ms":       m.WaitDurationMs,
		"thinking_duration_ms":   m.ThinkingDurationMs,
		"generation_duration_ms": m.GenerationDurationMs,
		"output_tokens":          m.OutputTokens,
		"input_tokens":           m.InputTokens,
		"estimated_cost_usd":     m.EstimatedCostUSD,
		"tokens_per_sec":         m.TokensPerSec,
		"had_thinking":           m.HadThinking,
		"tool_calls":             m.ToolCalls,
		"tool_duration_ms":       m.ToolDurationMs,
		"steps":                  m.Steps,
		"context_tokens":         m.ContextTokens,
		"context_window":         m.ContextWindow,
		"context_used_pct":       m.ContextUsedPct,
	}
}

func estimateTokens(text string) int {
	n := len([]rune(text))
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}
