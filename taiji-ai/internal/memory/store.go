package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Store reads and writes compact JSON memory indexes.
type Store struct {
	globalDir  string
	projectDir string
	mu         sync.Mutex
	cache      map[string]*scopedIndex
}

func NewStore(globalDir, projectDir string) *Store {
	return &Store{
		globalDir:  strings.TrimSpace(globalDir),
		projectDir: strings.TrimSpace(projectDir),
		cache:      map[string]*scopedIndex{},
	}
}

func (s *Store) ListAll() ([]Entry, error) {
	var out []Entry
	for _, scope := range []struct {
		label string
		dir   string
	}{
		{"local", s.projectDir},
		{"global", s.globalDir},
	} {
		if scope.dir == "" {
			continue
		}
		entries, err := s.listScopeEntries(scope.label, scope.dir)
		if err != nil {
			return nil, err
		}
		out = append(out, entries...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Scope == out[j].Scope {
			return out[i].ID < out[j].ID
		}
		return out[i].Scope == "local"
	})
	return out, nil
}

func (s *Store) Save(scope string, entry Entry) (Entry, error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return Entry{}, err
	}
	return s.saveScopeEntry(scopeLabel(scope), dir, entry)
}

func (s *Store) FindByID(id string) (Entry, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Entry{}, errIDRequired
	}
	all, err := s.ListAll()
	if err != nil {
		return Entry{}, err
	}
	for _, e := range all {
		if e.ID == id {
			return e, nil
		}
	}
	return Entry{}, errEntryNotFound
}

func (s *Store) dirForScope(scope string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "local":
		if s.projectDir == "" {
			return "", fmt.Errorf("project memory dir is not configured")
		}
		return s.projectDir, nil
	case "global":
		if s.globalDir == "" {
			return "", fmt.Errorf("global memory dir is not configured")
		}
		return s.globalDir, nil
	default:
		return "", fmt.Errorf("unknown memory scope %q (use local or global)", scope)
	}
}

func scopeLabel(scope string) string {
	if strings.EqualFold(strings.TrimSpace(scope), "global") {
		return "global"
	}
	return "local"
}

// LoadMarkdownFile parses a legacy memory markdown file.
func LoadMarkdownFile(path string) (Entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, err
	}
	content := string(raw)
	meta, body := parseFrontmatter(content)
	entry := Entry{
		ID:            strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Kind:          meta.Kind,
		Title:         strings.TrimSpace(meta.Title),
		Fingerprint:   strings.TrimSpace(meta.Fingerprint),
		Tags:          append([]string(nil), meta.Tags...),
		Keywords:      append([]string(nil), meta.Keywords...),
		RecallQueries: append([]string(nil), meta.RecallQueries...),
		Hits:          meta.Hits,
		Misses:        meta.Misses,
		LastVerified:  strings.TrimSpace(meta.LastVerified),
		RecallText:    strings.TrimSpace(meta.RecallText),
		Path:          path,
	}
	entry.VerifiedFix = extractSection(body, "Verified fix")
	if entry.VerifiedFix == "" {
		entry.VerifiedFix = extractSection(body, "Verified Fix")
	}
	entry.RevisionLog = extractSection(body, "Revision log")
	if entry.RevisionLog == "" {
		entry.RevisionLog = extractSection(body, "Revision Log")
	}
	if entry.Kind == "" {
		entry.Kind = KindInsight
	}
	return entry, nil
}

// LoadFile loads an entry from entries.json#id or a legacy markdown file.
func LoadFile(path string) (Entry, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Entry{}, fmt.Errorf("path is required")
	}
	if strings.Contains(path, "#") {
		parts := strings.SplitN(path, "#", 2)
		indexFile := parts[0]
		id := strings.TrimSpace(parts[1])
		raw, err := os.ReadFile(indexFile)
		if err != nil {
			return Entry{}, err
		}
		var index memoryIndex
		if err := json.Unmarshal(raw, &index); err != nil {
			return Entry{}, err
		}
		rec, ok := index.Entries[id]
		if !ok {
			return Entry{}, errEntryNotFound
		}
		scope := scopeForIndexFile(indexFile)
		return recordToEntry(rec, scope, indexFile), nil
	}
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		return LoadMarkdownFile(path)
	}
	return Entry{}, fmt.Errorf("unsupported memory path %q", path)
}

func scopeForIndexFile(indexFile string) string {
	clean := filepath.Clean(indexFile)
	parts := strings.Split(clean, string(os.PathSeparator))
	for i, part := range parts {
		if part == "memory" && i > 0 && parts[i-1] == ".taiji" {
			return "local"
		}
	}
	if strings.Contains(clean, string(os.PathSeparator)+"global"+string(os.PathSeparator)) {
		return "global"
	}
	return "local"
}

func parseFrontmatter(content string) (entryFrontmatter, string) {
	meta := entryFrontmatter{}
	if !strings.HasPrefix(content, "---\n") {
		return meta, content
	}
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) != 2 {
		return meta, content
	}
	_ = yaml.Unmarshal([]byte(parts[0]), &meta)
	return meta, parts[1]
}
