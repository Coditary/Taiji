package agent

import "strings"

type embeddedToolStreamFilter struct {
	buf     strings.Builder
	emitted int
	inner   func(string) error
}

func newEmbeddedToolStreamFilter(inner func(string) error) *embeddedToolStreamFilter {
	return &embeddedToolStreamFilter{inner: inner}
}

func (f *embeddedToolStreamFilter) Write(delta string) error {
	if delta == "" || f.inner == nil {
		return nil
	}
	f.buf.WriteString(delta)
	return f.flush(false)
}

func (f *embeddedToolStreamFilter) FlushRemaining() error {
	return f.flush(true)
}

// DiscardHeldSuffix drops any buffered tool-call suffix without emitting it.
func (f *embeddedToolStreamFilter) DiscardHeldSuffix() {
	f.emitted = len(f.buf.String())
}

func (f *embeddedToolStreamFilter) flush(final bool) error {
	s := f.buf.String()
	limit := len(s)
	if !final {
		limit = safeEmbeddedToolStreamLen(s)
	}
	if limit <= f.emitted {
		return nil
	}
	delta := s[f.emitted:limit]
	f.emitted = limit
	return f.inner(delta)
}

func safeEmbeddedToolStreamLen(s string) int {
	lower := strings.ToLower(s)
	idx := strings.Index(lower, embeddedToolCallMarker)
	if idx >= 0 {
		return idx
	}
	if hold := partialEmbeddedToolMarkerHoldLen(lower); hold >= 0 {
		return hold
	}
	return len(s)
}

func partialEmbeddedToolMarkerHoldLen(lower string) int {
	marker := strings.ToLower(embeddedToolCallMarker)
	for i := 1; i < len(marker); i++ {
		prefix := marker[:i]
		if strings.HasSuffix(lower, prefix) {
			return len(lower) - i
		}
	}
	return -1
}
