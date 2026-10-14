package agent

import (
	"testing"
	"time"

	"github.com/coditary/taiji/internal/provider"
)

func TestRunMetricsCollectorWaitVsGeneration(t *testing.T) {
	c := newRunMetricsCollector()

	c.OnCompletionStart()
	time.Sleep(40 * time.Millisecond)
	c.OnAnswerDelta()
	time.Sleep(25 * time.Millisecond)
	c.OnAnswerDelta()
	c.OnCompletionFinished()
	c.AddUsage(provider.Usage{OutputTokens: 20})

	m := c.Finalize("")
	if m.WaitDurationMs < 30 {
		t.Fatalf("WaitDurationMs = %d, want >= 30", m.WaitDurationMs)
	}
	if m.GenerationDurationMs < 20 {
		t.Fatalf("GenerationDurationMs = %d, want >= 20", m.GenerationDurationMs)
	}
	if m.WaitDurationMs <= m.GenerationDurationMs {
		t.Fatalf("wait=%d generation=%d, want wait > generation", m.WaitDurationMs, m.GenerationDurationMs)
	}
}

func TestRunMetricsCollectorGenerationSpeedExcludesThinking(t *testing.T) {
	c := newRunMetricsCollector()

	c.OnThinkingActivity()
	time.Sleep(20 * time.Millisecond)
	c.pauseThinking()

	c.OnAnswerDelta()
	time.Sleep(30 * time.Millisecond)
	c.OnAnswerDelta()
	c.AddUsage(provider.Usage{OutputTokens: 60})

	m := c.Finalize("")
	if m.HadThinking == false {
		t.Fatal("expected thinking")
	}
	if m.ThinkingDurationMs <= 0 {
		t.Fatalf("ThinkingDurationMs = %d", m.ThinkingDurationMs)
	}
	if m.GenerationDurationMs < 25 {
		t.Fatalf("GenerationDurationMs = %d", m.GenerationDurationMs)
	}
	if m.TokensPerSec <= 0 {
		t.Fatalf("TokensPerSec = %f", m.TokensPerSec)
	}
	totalGenSec := float64(m.GenerationDurationMs) / 1000
	wantTPS := float64(m.OutputTokens) / totalGenSec
	if m.TokensPerSec < wantTPS*0.5 || m.TokensPerSec > wantTPS*1.5 {
		t.Fatalf("TokensPerSec = %f, want near %f", m.TokensPerSec, wantTPS)
	}
}
