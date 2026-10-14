package memory

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	absPathPattern = regexp.MustCompile(`(?i)(/[A-Za-z0-9._~/-]+|(?:[A-Za-z]:\\)[^\s]+)`)
	relPathPattern = regexp.MustCompile(`(?i)\b(?:\./|\.\./)[^\s]+`)
	lineNumPattern = regexp.MustCompile(`(?i)(:[0-9]+(?:[:][0-9]+)?)\b`)
	uuidPattern    = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	hexPattern     = regexp.MustCompile(`(?i)\b0x[0-9a-f]+\b`)
	numPattern     = regexp.MustCompile(`\b[0-9]+\b`)
	spacePattern   = regexp.MustCompile(`\s+`)
)

// Fingerprint normalizes error text for deduplication and lookup.
func Fingerprint(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return ""
	}
	text = absPathPattern.ReplaceAllString(text, "<path>")
	text = relPathPattern.ReplaceAllString(text, "<path>")
	text = lineNumPattern.ReplaceAllString(text, ":<n>")
	text = uuidPattern.ReplaceAllString(text, "<id>")
	text = hexPattern.ReplaceAllString(text, "<hex>")
	text = numPattern.ReplaceAllString(text, "<n>")
	text = spacePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// MatchQuality scores how specifically a stored pattern matches computed text (0..1).
// Used to prefer narrow fingerprints over broad wildcards when several entries match.
func MatchQuality(pattern, computed string) float32 {
	rawPattern := strings.TrimSpace(strings.ToLower(pattern))
	computed = Fingerprint(computed)
	if rawPattern == "" || computed == "" || !MatchesFingerprint(pattern, computed) {
		return 0
	}
	normalizedPattern := Fingerprint(strings.ReplaceAll(strings.ReplaceAll(rawPattern, ":", " "), "*", ""))
	tokens := strings.Fields(normalizedPattern)
	if len(tokens) == 0 {
		return 0.1
	}
	matched := 0
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		if strings.Contains(computed, tok) {
			matched++
		}
	}
	quality := float32(matched) / float32(len(tokens))
	quality += float32(len(tokens)) * 0.01
	if !strings.HasSuffix(rawPattern, "*") {
		quality += 0.15
	}
	if strings.Count(rawPattern, "*") > 0 {
		quality -= float32(strings.Count(rawPattern, "*")) * 0.05
	}
	if quality < 0 {
		return 0
	}
	if quality > 1 {
		return 1
	}
	return quality
}

// MatchesFingerprint reports whether stored pattern matches a computed fingerprint.
// Patterns may use ":" as token separators and "*" for prefix matching.
func MatchesFingerprint(pattern, computed string) bool {
	rawPattern := strings.TrimSpace(strings.ToLower(pattern))
	computed = Fingerprint(computed)
	if rawPattern == "" || computed == "" {
		return false
	}
	normalizedPattern := Fingerprint(strings.ReplaceAll(strings.ReplaceAll(rawPattern, ":", " "), "*", ""))
	if normalizedPattern == "" {
		return false
	}
	if strings.HasSuffix(rawPattern, "*") {
		if strings.HasPrefix(computed, normalizedPattern) {
			return true
		}
		tokens := strings.Fields(normalizedPattern)
		for _, tok := range tokens {
			if tok == "" {
				continue
			}
			if !strings.Contains(computed, tok) {
				return false
			}
		}
		return len(tokens) > 0
	}
	return normalizedPattern == computed || strings.Contains(computed, normalizedPattern)
}

// SlugFromTitle builds a filesystem-safe entry id from a title or fingerprint.
func SlugFromTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return "entry"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '_' || r == '-' || r == '/' || r == ':':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "entry"
	}
	if len(out) > 80 {
		out = out[:80]
		out = strings.Trim(out, "-")
	}
	return out
}

// UniquePath returns path with numeric suffix if needed.
func UniquePath(dir, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "entry.md"
	}
	if filepath.Ext(base) != ".md" {
		base += ".md"
	}
	candidate := filepath.Join(dir, base)
	if _, err := filepath.Abs(candidate); err == nil {
		// keep going
	}
	for i := 0; i < 1000; i++ {
		path := candidate
		if i > 0 {
			ext := filepath.Ext(base)
			stem := strings.TrimSuffix(base, ext)
			path = filepath.Join(dir, stem+"-"+itoa(i)+ext)
		}
		if !fileExists(path) {
			return path
		}
	}
	return candidate
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [16]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
