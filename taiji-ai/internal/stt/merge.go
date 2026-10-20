package stt

import (
	"strings"
	"unicode"
)

// AppendTranscript merges a new partial transcription into accumulated text.
func AppendTranscript(accumulated, chunk string) string {
	chunk = strings.TrimSpace(chunk)
	if chunk == "" {
		return accumulated
	}
	if accumulated == "" {
		return chunk
	}
	lowAcc := strings.ToLower(accumulated)
	lowChunk := strings.ToLower(chunk)
	if strings.HasPrefix(lowChunk, lowAcc) {
		return chunk
	}
	if strings.HasSuffix(lowAcc, lowChunk) {
		return accumulated
	}
	// Overlap: new chunk extends the tail of accumulated.
	for n := len(chunk); n > 0; n-- {
		suffix := strings.ToLower(chunk[:n])
		if len(suffix) < 3 {
			break
		}
		if strings.HasSuffix(lowAcc, suffix) {
			rest := strings.TrimSpace(chunk[n:])
			if rest == "" {
				return accumulated
			}
			return accumulated + needSpace(accumulated) + rest
		}
	}
	return accumulated + needSpace(accumulated) + chunk
}

func needSpace(s string) string {
	if s == "" {
		return ""
	}
	if unicode.IsSpace(rune(s[len(s)-1])) {
		return ""
	}
	return " "
}
