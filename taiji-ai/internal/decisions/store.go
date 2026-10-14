package decisions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coditary/taiji/internal/decision"
	"gopkg.in/yaml.v3"
)

var profileNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

type writeMeta struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description,omitempty"`
	When        string                 `yaml:"when,omitempty"`
	Questions   map[string]writeQuestion `yaml:"questions"`
}

type writeQuestion struct {
	Type         string   `yaml:"type"`
	Instructions string   `yaml:"instructions"`
	Criteria     []string `yaml:"criteria,omitempty"`
}

// NormalizeProfileName lowercases and validates a decision profile name.
func NormalizeProfileName(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", "-")
	if name == "" {
		return "", fmt.Errorf("profile name is required")
	}
	if !profileNamePattern.MatchString(name) {
		return "", fmt.Errorf("profile name %q must be lowercase letters, numbers, and hyphens", name)
	}
	return name, nil
}

// SaveProfile writes .taiji/decisions/<name>/DECISION.md.
func SaveProfile(baseDir string, profile Profile) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", fmt.Errorf("decisions directory is not configured")
	}
	name, err := NormalizeProfileName(profile.Name)
	if err != nil {
		return "", err
	}
	if len(profile.Questions) == 0 {
		return "", fmt.Errorf("profile %q needs at least one question", name)
	}

	questions := make(map[string]writeQuestion, len(profile.Questions))
	for _, q := range profile.Questions {
		id := strings.TrimSpace(q.ID)
		if id == "" {
			return "", fmt.Errorf("question id is required")
		}
		wq := writeQuestion{
			Type:         string(q.Type),
			Instructions: strings.TrimSpace(q.Instructions),
		}
		if len(q.Criteria) > 0 {
			wq.Criteria = append([]string(nil), q.Criteria...)
		}
		questions[id] = wq
	}

	meta := writeMeta{
		Name:        name,
		Description: strings.TrimSpace(profile.Description),
		When:        strings.TrimSpace(profile.When),
		Questions:   questions,
	}
	yamlBytes, err := yaml.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("marshal profile %q: %w", name, err)
	}

	body := strings.TrimSpace(profile.Body)
	if body == "" {
		body = "# " + name + " decision\n\nSaved by the agent for reuse via decide({\"profile\": \"" + name + "\", \"text\": \"…\"})."
	}

	dir := filepath.Join(baseDir, name)
	path := filepath.Join(dir, decisionFileName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	content := "---\n" + string(yamlBytes) + "---\n\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ProfileFromQuestions builds a saveable profile from ad-hoc decide args.
func ProfileFromQuestions(name, description, when string, questions []decision.Question, body string) Profile {
	return Profile{
		Name:        name,
		Description: description,
		When:        when,
		Questions:   append([]decision.Question(nil), questions...),
		Body:        body,
	}
}
