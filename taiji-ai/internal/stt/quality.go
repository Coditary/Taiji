package stt

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseQuality maps wuji-style quality names to beam size (0 = backend default).
func ParseQuality(raw string) (int, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" {
		return 0, nil
	}
	switch normalized {
	case "fast":
		return 1, nil
	case "balanced":
		return 3, nil
	case "careful":
		return 5, nil
	default:
		n, err := strconv.Atoi(normalized)
		if err != nil {
			return 0, fmt.Errorf("quality must be fast, balanced, careful, or 1–5")
		}
		if n < 1 || n > 5 {
			return 0, fmt.Errorf("quality beam width must be 1–5")
		}
		return n, nil
	}
}
