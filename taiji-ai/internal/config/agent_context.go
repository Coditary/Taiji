package config

const defaultMaxContextChars = 160000

// MaxContextCharsOrDefault returns the history size threshold for auto-compaction (0 disables).
func (a AgentConfig) MaxContextCharsOrDefault() int {
	if a.MaxContextChars == nil {
		return defaultMaxContextChars
	}
	return *a.MaxContextChars
}
