package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill is a loaded SKILL.md document.
type Skill struct {
	Name                   string
	Description            string
	Body                   string
	Path                   string
	DisableModelInvocation bool
}

// Loader scans skill directories for SKILL.md files.
type Loader struct {
	dirs []string
}

// NewLoader creates a skill loader.
func NewLoader(dirs []string) *Loader {
	return &Loader{dirs: dirs}
}

// LoadAll returns all skills found in configured directories.
func (l *Loader) LoadAll() ([]Skill, error) {
	var out []Skill
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
			path := filepath.Join(dir, e.Name(), "SKILL.md")
			skill, err := LoadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			if seen[skill.Name] {
				continue
			}
			seen[skill.Name] = true
			out = append(out, skill)
		}
	}
	return out, nil
}

// LoadFile parses a SKILL.md file with optional YAML frontmatter.
func LoadFile(path string) (Skill, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}
	content := string(raw)
	name := filepath.Base(filepath.Dir(path))
	meta := parseFrontmatter(content)
	if meta.name != "" {
		name = meta.name
	}
	return Skill{
		Name:                   name,
		Description:            meta.description,
		Body:                   strings.TrimSpace(meta.body),
		Path:                   path,
		DisableModelInvocation: meta.disableModelInvocation,
	}, nil
}

type frontmatter struct {
	name                   string
	description            string
	body                   string
	disableModelInvocation bool
}

func parseFrontmatter(content string) frontmatter {
	meta := frontmatter{body: content}
	if !strings.HasPrefix(content, "---\n") {
		return meta
	}
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) != 2 {
		return meta
	}
	meta.body = parts[1]
	for _, line := range strings.Split(parts[0], "\n") {
		line = strings.TrimPrefix(line, "---")
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			meta.name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
		if strings.HasPrefix(line, "description:") {
			meta.description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
		if _, val, ok := frontmatterBool(line, "disable-model-invocation"); ok {
			meta.disableModelInvocation = val
		} else if _, val, ok := frontmatterBool(line, "disable_model_invocation"); ok {
			meta.disableModelInvocation = val
		}
	}
	return meta
}

func frontmatterBool(line, key string) (string, bool, bool) {
	if !strings.HasPrefix(line, key+":") {
		return "", false, false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(line, key+":"))
	switch strings.ToLower(raw) {
	case "true", "yes", "1":
		return key, true, true
	case "false", "no", "0", "":
		return key, false, true
	default:
		return key, false, true
	}
}

// BuildSystemAppendix returns skill text to append to the system prompt.
func BuildSystemAppendix(skills []Skill, names []string) string {
	if len(names) == 0 || len(skills) == 0 {
		return ""
	}
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var parts []string
	for _, s := range skills {
		if !want[s.Name] {
			continue
		}
		parts = append(parts, "## Skill: "+s.Name+"\n\n"+s.Body)
	}
	return strings.Join(parts, "\n\n")
}
