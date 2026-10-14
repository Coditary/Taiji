package builtin

import (
	"context"
	"fmt"

	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/tools"
)

// EnableMemoryOrchestration registers memory tools.
func EnableMemoryOrchestration(reg *tools.Registry, svc *memory.Service) {
	if reg == nil || svc == nil || !svc.Enabled() {
		return
	}
	reg.Register(MemoryRecall{svc: svc})
	reg.Register(MemoryRemember{svc: svc})
	reg.Register(MemoryRevise{svc: svc})
}

type MemoryRecall struct {
	svc *memory.Service
}

func (MemoryRecall) Name() string { return "memory_recall" }

func (t MemoryRecall) Description() string {
	return "Search Taiji memory for prior verified experience. Always set kind to match the situation — see the memory kinds catalog in the system prompt."
}

func (t MemoryRecall) Parameters() map[string]any {
	kindDesc := "Required context filter. See memory kinds catalog in system prompt."
	if catalog := t.svc.KindCatalog(); catalog != nil {
		kindDesc = "Required context filter: " + catalog.KindNamesDescription() + ". See memory kinds catalog in system prompt."
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Natural language query or error text",
			},
			"kind": map[string]any{
				"type":        "string",
				"description": kindDesc,
			},
			"fingerprint": map[string]any{
				"type":        "string",
				"description": "Optional normalized error fingerprint",
			},
			"tool_name": map[string]any{
				"type":        "string",
				"description": "Optional tool name for context",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "Maximum matches to return (default 5)",
			},
		},
		"required": []string{"query"},
	}
}

func (t MemoryRecall) Run(ctx context.Context, args map[string]any) (string, error) {
	topK := intArg(args, "top_k")
	if topK <= 0 {
		topK = 5
	}
	entries, err := t.svc.Recall(ctx, memory.RecallRequest{
		Query:       stringArg(args, "query"),
		Kind:        stringArg(args, "kind"),
		Fingerprint: stringArg(args, "fingerprint"),
		ToolName:    stringArg(args, "tool_name"),
		TopK:        topK,
	})
	if err != nil {
		return "", err
	}
	return memory.FormatRecallResults(entries), nil
}

type MemoryRemember struct {
	svc *memory.Service
}

func (MemoryRemember) Name() string { return "memory_remember" }

func (t MemoryRemember) Description() string {
	return "Store a new verified memory entry after success. Pick kind from the memory kinds catalog. For preferences and persona, add keywords and recall_queries with likely user questions and synonyms."
}

func (t MemoryRemember) Parameters() map[string]any {
	kindDesc := "Memory kind from the catalog in the system prompt"
	if catalog := t.svc.KindCatalog(); catalog != nil {
		kindDesc = catalog.KindNamesDescription()
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind": map[string]any{
				"type":        "string",
				"description": kindDesc,
			},
			"recall_text": map[string]any{
				"type":        "string",
				"description": "What to remember (preferred field name). Required unless query/text/content is set.",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Alias for recall_text",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Alias for recall_text — same meaning",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Alias for recall_text — same meaning",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Short title (optional — derived from recall text when omitted)",
			},
			"verified_fix": map[string]any{
				"type":        "string",
				"description": "Verified steps or knowledge to apply. Required for error/pattern/workflow kinds. For preference/persona/insight, the fact/value (e.g. name); defaults to recall_text when omitted.",
			},
			"fingerprint": map[string]any{
				"type":        "string",
				"description": "Optional normalized fingerprint (auto for errors)",
			},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional tags",
			},
			"keywords": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional recall keywords and synonyms (e.g. name, heißt, vorname)",
			},
			"recall_queries": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional alternative user questions that should recall this entry",
			},
			"scope": map[string]any{
				"type":        "string",
				"description": "local (default) or global",
			},
		},
		"required": []string{"kind"},
	}
}

func (t MemoryRemember) Run(ctx context.Context, args map[string]any) (string, error) {
	entry, err := t.svc.Remember(ctx, memory.RememberRequest{
		Kind:        stringArg(args, "kind"),
		Title:       stringArg(args, "title"),
		RecallText:  memoryRecallTextArg(args),
		VerifiedFix: memoryVerifiedFixArg(args),
		Fingerprint: stringArg(args, "fingerprint"),
		Tags:          stringSliceArg(args["tags"]),
		Keywords:      stringSliceArg(args["keywords"]),
		RecallQueries: stringSliceArg(args["recall_queries"]),
		Scope:         stringArg(args, "scope"),
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("remembered id=%s kind=%s scope=%s path=%s", entry.ID, entry.Kind, entry.Scope, entry.Path), nil
}

type MemoryRevise struct {
	svc *memory.Service
}

func (MemoryRevise) Name() string { return "memory_revise" }

func (MemoryRevise) Description() string {
	return "Update an existing memory entry after a better fix, a miss, or new context."
}

func (MemoryRevise) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Memory entry id (filename stem)",
			},
			"recall_text": map[string]any{
				"type":        "string",
				"description": "Optional updated recall text",
			},
			"verified_fix": map[string]any{
				"type":        "string",
				"description": "Optional updated verified fix",
			},
			"revision_note": map[string]any{
				"type":        "string",
				"description": "What changed or what was missing",
			},
			"outcome": map[string]any{
				"type":        "string",
				"description": "Optional outcome: success/hit or miss/failure",
			},
			"keywords": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional keywords/synonyms to add",
			},
			"recall_queries": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional alternative user questions to add",
			},
		},
		"required": []string{"id", "revision_note"},
	}
}

func (t MemoryRevise) Run(ctx context.Context, args map[string]any) (string, error) {
	entry, err := t.svc.Revise(ctx, memory.ReviseRequest{
		ID:            stringArg(args, "id"),
		RecallText:    memoryRecallTextArg(args),
		VerifiedFix:   memoryVerifiedFixArg(args),
		RevisionNote:  stringArg(args, "revision_note"),
		Outcome:       stringArg(args, "outcome"),
		Keywords:      stringSliceArg(args["keywords"]),
		RecallQueries: stringSliceArg(args["recall_queries"]),
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("revised id=%s kind=%s scope=%s hits=%d misses=%d path=%s",
		entry.ID, entry.Kind, entry.Scope, entry.Hits, entry.Misses, entry.Path), nil
}
