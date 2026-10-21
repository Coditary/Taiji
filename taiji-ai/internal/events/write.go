package events

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// WriteFile replaces path with a JSONL file containing evts (oldest first).
func WriteFile(path string, evts []Event) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("events path is empty")
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, ev := range evts {
		raw, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(f, "%s\n", raw); err != nil {
			return err
		}
	}
	return nil
}
