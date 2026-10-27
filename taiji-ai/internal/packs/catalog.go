package packs

// PackSummary describes one pack folder for settings / TUI.
type PackSummary struct {
	Name      string
	NodeCount int
}

// Summarize groups loaded nodes by pack name.
func Summarize(nodes []PackNode) []PackSummary {
	counts := map[string]int{}
	for _, n := range nodes {
		if n.Legacy != LegacyNone {
			continue
		}
		name := n.PackName
		if name == "" {
			continue
		}
		counts[name]++
	}
	out := make([]PackSummary, 0, len(counts))
	for name, c := range counts {
		out = append(out, PackSummary{Name: name, NodeCount: c})
	}
	sortSummaries(out)
	return out
}

// MentionEntry is one selectable pack node for @-completion.
type MentionEntry struct {
	ID    string
	Label string
	Pack  string
}

// ListMentions returns mention candidates from active nodes.
func ListMentions(nodes []PackNode, allow []string) []MentionEntry {
	var out []MentionEntry
	for _, n := range nodes {
		if !NodeAllowed(n, allow) {
			continue
		}
		if n.Legacy == LegacyAgentsMD {
			continue
		}
		out = append(out, MentionEntry{
			ID:    n.NodeID(),
			Label: n.DisplayName(),
			Pack:  n.PackName,
		})
	}
	return out
}

func sortSummaries(summaries []PackSummary) {
	for i := 0; i < len(summaries); i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[j].Name < summaries[i].Name {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}
}
