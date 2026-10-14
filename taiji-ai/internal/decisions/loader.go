package decisions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coditary/taiji/internal/decision"
	"gopkg.in/yaml.v3"
)

const decisionFileName = "DECISION.md"

// Loader scans decision profile directories.
type Loader struct {
	dirs []string
}

// NewLoader creates a profile loader.
func NewLoader(dirs []string) *Loader {
	return &Loader{dirs: dirs}
}

type fileMeta struct {
	Name                   string         `yaml:"name"`
	Description            string         `yaml:"description"`
	When                   string         `yaml:"when"`
	Model                  string         `yaml:"model"`
	Driver                 string         `yaml:"driver"`
	DisableModelInvocation bool           `yaml:"disable_model_invocation"`
	Questions              map[string]any `yaml:"questions"`
}

// LoadAll returns all profiles from configured directories.
func (l *Loader) LoadAll() ([]Profile, error) {
	var out []Profile
	seen := map[string]bool{}
	for _, dir := range l.dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			path := filepath.Join(dir, e.Name(), decisionFileName)
			profile, err := LoadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			key := lookupKey(profile.Name)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, profile)
		}
	}
	return out, nil
}

// LoadFile parses a DECISION.md profile.
func LoadFile(path string) (Profile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	content := string(raw)
	body := content
	var meta fileMeta

	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content, "\n---\n", 2)
		if len(parts) == 2 {
			yamlBlock := strings.TrimPrefix(parts[0], "---")
			if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
				return Profile{}, fmt.Errorf("parse decision frontmatter: %w", err)
			}
			body = parts[1]
		}
	}

	name := strings.TrimSpace(meta.Name)
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}
	questions, err := decision.ParseQuestions(meta.Questions)
	if err != nil {
		return Profile{}, fmt.Errorf("profile %q: %w", name, err)
	}

	return Profile{
		Name:                   name,
		Description:            strings.TrimSpace(meta.Description),
		When:                   strings.TrimSpace(meta.When),
		Model:                  strings.TrimSpace(meta.Model),
		Driver:                 strings.TrimSpace(meta.Driver),
		Questions:              questions,
		Body:                   strings.TrimSpace(body),
		Path:                   path,
		DisableModelInvocation: meta.DisableModelInvocation,
	}, nil
}
