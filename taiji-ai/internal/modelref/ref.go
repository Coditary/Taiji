package modelref

import (
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// Display strips Wuji llama storage prefixes (ollama/… and .gguf) for UI and session storage.
func Display(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	const ollamaPrefix = "ollama/"
	if strings.HasPrefix(ref, ollamaPrefix) {
		ref = ref[len(ollamaPrefix):]
	}
	if len(ref) > 5 && strings.EqualFold(ref[len(ref)-5:], ".gguf") {
		ref = ref[:len(ref)-5]
	}
	return ref
}

// Expand maps a user-facing model name back to a Wuji llama model ref when possible.
func Expand(input string, candidates []string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if c == input {
			return c
		}
		if Display(c) == input {
			return c
		}
	}
	if !strings.Contains(input, "/") && !strings.HasSuffix(strings.ToLower(input), ".gguf") {
		try := "ollama/" + input + ".gguf"
		for _, c := range candidates {
			if c == try {
				return c
			}
		}
		if len(candidates) == 0 {
			return try
		}
	}
	return input
}

// Candidates collects canonical model refs for a provider from config.
func Candidates(cfg *taijicfg.Config, providerID string, extra ...string) []string {
	var out []string
	add := func(m string) {
		m = strings.TrimSpace(m)
		if m == "" {
			return
		}
		for _, existing := range out {
			if existing == m {
				return
			}
		}
		out = append(out, m)
	}
	for _, m := range extra {
		add(m)
	}
	if cfg == nil || cfg.Providers == nil {
		return out
	}
	if pc, ok := cfg.Providers[providerID]; ok {
		add(pc.Model)
	}
	return out
}
