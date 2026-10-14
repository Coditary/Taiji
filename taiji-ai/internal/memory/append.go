package memory

import (
	"fmt"
	"strings"
)

// SystemPromptAppendix tells the agent how Taiji memory works.
func SystemPromptAppendix(s *Service) string {
	if s == nil || !s.Enabled() {
		return ""
	}
	catalogSection := ""
	if s.kinds != nil {
		catalogSection = s.kinds.BuildCatalogAppendix()
	}
	body := strings.TrimSpace(fmt.Sprintf(`## Memory

Taiji remembers verified experience across sessions. Memory is separate from skills (manual workflows) and generic RAG docs.

Store: %s

%s

### Tools

- memory_recall — search memory (BM25 + synonyms + RAG). **Always pass kind** matching the situation.
- memory_remember — store new verified knowledge after success. Pick kind from the catalog above.
- memory_revise — update an existing entry (preferred over duplicating).

### Lookup before claiming ignorance

If the user asks about something you do not see in the current chat (name, preferences, past fixes, project conventions):
1. **Call memory_recall first** with the matching kind and the user's question as query.
2. **Only after** recall returns no match, say you could not find it — and ask the user or offer to store it with memory_remember.
3. **Never** say "I don't know your name" or "tell me first" without calling memory_recall when memory is enabled.
4. When the user says "remember", "erinnere dich", or "use the memory tool": call memory_recall immediately (do not ask them to repeat facts that may already be stored).

### Workflow

1. **Failure:** use error/anti_pattern recall only. Verify match before applying fix.
2. **User question:** pick the kind from the catalog (e.g. preference for user name/likes, persona for assistant name/role).
3. **After verified success:** reflect briefly, then memory_remember or memory_revise.
4. **Reflection:** what was the root cause? was recall correct? what was missing? update entry if needed.
5. **Never store:** secrets, one-off state, things that belong in README/AGENTS.md, unverified guesses.

### Precision rules

- Wrong kind is worse than no recall. Do not apply error fixes to preference questions.
- Prefer revising one entry over creating duplicates.
- If recall misses, say so honestly — then ask or store what you learned.`,
		s.DescribeDirs(), catalogSection))
	return body
}
