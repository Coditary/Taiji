package session

import (
	"encoding/json"
	"strings"
)

func parseLoreEntries(raw json.RawMessage) []charaLoreEntry {
	if len(raw) == 0 {
		return nil
	}
	var book charaLoreBook
	if err := json.Unmarshal(raw, &book); err == nil && len(book.Entries) > 0 {
		return filterEnabledLore(book.Entries)
	}
	var wrapper struct {
		Entries json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.Entries) > 0 {
		return parseLoreEntryListOrMap(wrapper.Entries)
	}
	return parseLoreEntryListOrMap(raw)
}

func parseLoreEntryListOrMap(raw json.RawMessage) []charaLoreEntry {
	var list []charaLoreEntry
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		return filterEnabledLore(list)
	}
	var asMap map[string]charaLoreEntry
	if err := json.Unmarshal(raw, &asMap); err == nil && len(asMap) > 0 {
		var out []charaLoreEntry
		for _, e := range asMap {
			out = append(out, e)
		}
		return filterEnabledLore(out)
	}
	return nil
}

func filterEnabledLore(entries []charaLoreEntry) []charaLoreEntry {
	var out []charaLoreEntry
	for _, e := range entries {
		if loreEntryDisabled(e) {
			continue
		}
		if strings.TrimSpace(e.Content) == "" {
			continue
		}
		out = append(out, e)
	}
	return out
}

func loreEntryDisabled(e charaLoreEntry) bool {
	if e.Enabled == nil {
		return false
	}
	return !*e.Enabled
}
