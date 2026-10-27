package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/session"
)

type SearchConversations struct{}

func (SearchConversations) Name() string { return "search_conversations" }

func (SearchConversations) Description() string {
	return "Search prior chat sessions on this project for messages matching keywords (all keywords must match)."
}

func (SearchConversations) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Keywords to find in user/assistant messages",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Max hits (default 20)",
			},
		},
		"required": []string{"query"},
	}
}

func (SearchConversations) Run(ctx context.Context, args map[string]any) (string, error) {
	query := strings.TrimSpace(stringArg(args, "query"))
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	limit := intArg(args, "limit")
	root := workspaceProjectRoot()
	if root == "" {
		root = "."
	}
	if resolveTaijiToolsBinary() != "" {
		out, err := runTaijiTools(ctx, "sessions-search", "--root", root, "--query", query, "--max", fmt.Sprintf("%d", limit))
		if err == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out), nil
		}
	}
	hits, err := session.SearchConversations(root, query, limit)
	if err != nil {
		return "", err
	}
	if len(hits) == 0 {
		return "No matching conversations.", nil
	}
	var lines []string
	for _, h := range hits {
		lines = append(lines, fmt.Sprintf("- session %s (%s) [%s]\n  %s", h.SessionID, h.SessionName, h.Role, h.Snippet))
	}
	return strings.Join(lines, "\n\n"), nil
}
