package events

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// ReadFile loads all events from a JSONL file (oldest first).
func ReadFile(path string) ([]Event, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	if err := scanner.Err(); err != nil {
		return out, err
	}
	return out, nil
}
