package agentsdoc

import (
	"os"
	"path/filepath"
	"strings"
)

const fileName = "AGENTS.md"

// Load reads AGENTS.md from dir when present.
// Returns found=false when the file does not exist or is empty.
func Load(dir string) (content string, found bool, err error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", false, nil
	}
	path := filepath.Join(dir, fileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	content = strings.TrimSpace(string(raw))
	if content == "" {
		return "", false, nil
	}
	return content, true, nil
}

// Path returns the AGENTS.md path under dir.
func Path(dir string) string {
	return filepath.Join(strings.TrimSpace(dir), fileName)
}

// SystemAppendix formats AGENTS.md content for the system prompt.
func SystemAppendix(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	return "## Project instructions (AGENTS.md)\n\n" + content
}
