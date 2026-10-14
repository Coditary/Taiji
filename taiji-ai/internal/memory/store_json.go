package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	entriesFileName    = "entries.json"
	memoryIndexVersion = 1
	ragIndexSubdir     = ".rag-index"
)

type memoryIndex struct {
	Version int                    `json:"version"`
	Entries map[string]entryRecord `json:"entries"`
}

type entryRecord struct {
	ID            string   `json:"id"`
	Kind          Kind     `json:"kind"`
	Title         string   `json:"title"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	RecallQueries []string `json:"recall_queries,omitempty"`
	Hits          int      `json:"hits,omitempty"`
	Misses        int      `json:"misses,omitempty"`
	LastVerified  string   `json:"last_verified,omitempty"`
	RecallText    string   `json:"recall_text"`
	VerifiedFix   string   `json:"verified_fix,omitempty"`
	RevisionLog   string   `json:"revision_log,omitempty"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
}

type scopedIndex struct {
	scope  string
	dir    string
	index  memoryIndex
	loaded bool
}

func (s *Store) listScopeEntries(scope, dir string) ([]Entry, error) {
	idx, err := s.loadScopeIndex(scope, dir)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, 0, len(idx.index.Entries))
	for _, rec := range idx.index.Entries {
		out = append(out, recordToEntry(rec, scope, indexPath(dir)))
	}
	return out, nil
}

func (s *Store) loadScopeIndex(scope, dir string) (*scopedIndex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadScopeIndexLocked(scope, dir)
}

func (s *Store) loadScopeIndexLocked(scope, dir string) (*scopedIndex, error) {
	if s == nil {
		return nil, fmt.Errorf("memory store is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	key := scopeKey(scope, dir)
	if cached, ok := s.cache[key]; ok && cached.loaded {
		return cached, nil
	}

	idx := &scopedIndex{
		scope: scope,
		dir:   dir,
		index: memoryIndex{
			Version: memoryIndexVersion,
			Entries: map[string]entryRecord{},
		},
	}

	path := indexPath(dir)
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &idx.index); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if idx.index.Entries == nil {
			idx.index.Entries = map[string]entryRecord{}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := importLegacyMarkdown(dir, idx); err != nil {
		return nil, err
	}

	idx.loaded = true
	s.cache[key] = idx
	return idx, nil
}

func (s *Store) saveScopeEntry(scope, dir string, entry Entry) (Entry, error) {
	idx, err := s.loadScopeIndex(scope, dir)
	if err != nil {
		return Entry{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(entry.ID)
	if id == "" {
		slug := SlugFromTitle(entry.Title)
		if slug == "" {
			slug = "entry"
		}
		id = uniqueID(idx.index.Entries, slug)
	}

	rec := entryToRecord(entry, id)
	rec.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	idx.index.Entries[id] = rec

	if err := writeMemoryIndex(dir, idx.index); err != nil {
		return Entry{}, err
	}

	out := recordToEntry(rec, scope, indexPath(dir))
	if err := writeRAGIndexFile(dir, out); err != nil {
		return out, fmt.Errorf("saved entry but rag index failed: %w", err)
	}
	return out, nil
}

func writeMemoryIndex(dir string, index memoryIndex) error {
	if index.Version == 0 {
		index.Version = memoryIndexVersion
	}
	if index.Entries == nil {
		index.Entries = map[string]entryRecord{}
	}
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	path := indexPath(dir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func indexPath(dir string) string {
	return filepath.Join(dir, entriesFileName)
}

func scopeKey(scope, dir string) string {
	return scope + "\x00" + filepath.Clean(dir)
}

func entryToRecord(entry Entry, id string) entryRecord {
	return entryRecord{
		ID:            id,
		Kind:          entry.Kind,
		Title:         entry.Title,
		Fingerprint:   entry.Fingerprint,
		Tags:          append([]string(nil), entry.Tags...),
		Keywords:      append([]string(nil), entry.Keywords...),
		RecallQueries: append([]string(nil), entry.RecallQueries...),
		Hits:          entry.Hits,
		Misses:        entry.Misses,
		LastVerified:  entry.LastVerified,
		RecallText:    entry.RecallText,
		VerifiedFix:   entry.VerifiedFix,
		RevisionLog:   entry.RevisionLog,
	}
}

func recordToEntry(rec entryRecord, scope, indexFile string) Entry {
	kind := rec.Kind
	if kind == "" {
		kind = KindInsight
	}
	return Entry{
		ID:            rec.ID,
		Kind:          kind,
		Title:         rec.Title,
		Fingerprint:   rec.Fingerprint,
		Tags:          append([]string(nil), rec.Tags...),
		Keywords:      append([]string(nil), rec.Keywords...),
		RecallQueries: append([]string(nil), rec.RecallQueries...),
		Hits:          rec.Hits,
		Misses:        rec.Misses,
		LastVerified:  rec.LastVerified,
		RecallText:    rec.RecallText,
		VerifiedFix:   rec.VerifiedFix,
		RevisionLog:   rec.RevisionLog,
		Scope:         scope,
		Path:          indexFile + "#" + rec.ID,
	}
}

func uniqueID(entries map[string]entryRecord, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "entry"
	}
	if _, ok := entries[base]; !ok {
		return base
	}
	for i := 1; i < 10000; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := entries[candidate]; !ok {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func importLegacyMarkdown(dir string, idx *scopedIndex) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	imported := false
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, file.Name())
		entry, err := LoadMarkdownFile(path)
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		id = uniqueID(idx.index.Entries, id)
		idx.index.Entries[id] = entryToRecord(entry, id)
		imported = true
		_ = os.Remove(path)
	}
	if !imported {
		return nil
	}
	return writeMemoryIndex(dir, idx.index)
}

func writeRAGIndexFile(dir string, entry Entry) error {
	ragDir := filepath.Join(dir, ragIndexSubdir)
	if err := os.MkdirAll(ragDir, 0o755); err != nil {
		return err
	}
	payload := map[string]any{
		"id":             entry.ID,
		"kind":           entry.Kind,
		"title":          entry.Title,
		"fingerprint":    entry.Fingerprint,
		"tags":           entry.Tags,
		"keywords":       entry.Keywords,
		"recall_queries": entry.RecallQueries,
		"recall_text":    entry.RecallText,
		"verified_fix":   entry.VerifiedFix,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	path := filepath.Join(ragDir, entry.ID+".json")
	return os.WriteFile(path, raw, 0o644)
}

func ragIndexPathForEntry(entryPath string) string {
	entryPath = strings.TrimSpace(entryPath)
	if entryPath == "" {
		return ""
	}
	if idx := strings.Index(entryPath, "#"); idx >= 0 {
		dir := filepath.Dir(entryPath[:idx])
		id := strings.TrimSpace(entryPath[idx+1:])
		if id == "" {
			return ""
		}
		return filepath.Join(dir, ragIndexSubdir, id+".json")
	}
	return entryPath
}
