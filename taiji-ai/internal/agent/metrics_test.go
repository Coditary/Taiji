package agent

import (
	"testing"
	"time"

	"github.com/coditary/taiji/internal/provider"
)

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
