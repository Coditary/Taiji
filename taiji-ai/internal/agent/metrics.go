package agent

import (
	"strings"
	"time"

	"github.com/coditary/taiji/internal/provider"
)

// RunMetrics summarizes timing and throughput for one user request.
type RunMetrics struct {
	TotalDurationMs      int64
	ThinkingDurationMs   int64
	OutputTokens         int
	GenerationDurationMs int64
	TokensPerSec         float64
	HadThinking          bool
}

type runMetricsCollector struct {
	start time.Time

	thinkingDuration time.Duration
	thinkingStart    time.Time
	inThinking       bool

	genStart       time.Time
	genEnd         time.Time
	inGeneration   bool
	outputTokens   int
	lastProviderTPS float64
	finished       bool
}

func newRunMetricsCollector() *runMetricsCollector {
	return &runMetricsCollector{start: time.Now()}
}

func (c *runMetricsCollector) OnThinkingActivity() {
	if c == nil || c.finished {
		return
	}
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
	if u.OutputTokens > 0 {
		c.outputTokens += u.OutputTokens
	}
	if u.TokensPerSec > 0 {
		c.lastProviderTPS = u.TokensPerSec
	}
}

func (c *runMetricsCollector) Finalize(finalAnswer string) RunMetrics {
	if c == nil {
		return RunMetrics{}
	}
	c.finished = true
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
		ThinkingDurationMs:   c.thinkingDuration.Milliseconds(),
		OutputTokens:         c.outputTokens,
		GenerationDurationMs: genDur.Milliseconds(),
		TokensPerSec:         tps,
		HadThinking:          c.thinkingDuration > 0,
	}
}

func estimateTokens(text string) int {
	n := len([]rune(text))
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}
