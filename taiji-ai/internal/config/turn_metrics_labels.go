package config

import "github.com/coditary/taiji/internal/uilocale"

// TurnMetricsLabels are suffixes and templates for per-turn timing lines (TUI locale).
type TurnMetricsLabels struct {
	Total      string
	Wait       string
	Thought    string
	Generation string
	Tok        string
	TokIn      string
	TokPerSec  string // fmt.Sprintf template with one float, e.g. "%.1f tok/s"
	ToolOne    string
	ToolsMany  string // fmt.Sprintf with count
	ToolTime   string
	StepOne    string
	StepsMany  string
	Ctx        string
	CtxPair    string // two token counts
	CtxPct     string // one float percent
}

func TurnMetricsLabelsForLocale(locale string) TurnMetricsLabels {
	id := NormalizeUILocale(locale)
	m := func(key string) string {
		return uilocale.Lookup(id, "tools", "metrics."+key)
	}
	return TurnMetricsLabels{
		Total:      m("total"),
		Wait:       m("wait"),
		Thought:    m("thought"),
		Generation: m("generation"),
		Tok:        m("tok"),
		TokIn:      m("tok_in"),
		TokPerSec:  m("tok_per_sec"),
		ToolOne:    m("tool_one"),
		ToolsMany:  m("tools_many"),
		ToolTime:   m("tool_time"),
		StepOne:    m("step_one"),
		StepsMany:  m("steps_many"),
		Ctx:        m("ctx"),
		CtxPair:    m("ctx_pair"),
		CtxPct:     m("ctx_pct"),
	}
}
