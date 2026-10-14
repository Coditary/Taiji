package memory

import (
	"fmt"
	"os"
	"sort"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"gopkg.in/yaml.v3"
)

// KindDefinition describes one memory kind for agents and validation.
type KindDefinition struct {
	Name                string
	Description         string
	StoreWhen           string
	RecallWhen          string
	RequiresVerifiedFix bool
	FailureKind         bool
	AutoFingerprint     bool
	EnrichName          bool
	RecallHints         []string
	StoreHints          []string
}

type kindFile struct {
	Kinds []kindFileEntry `yaml:"kinds"`
}

type kindFileEntry struct {
	Name                string   `yaml:"name"`
	Description         string   `yaml:"description,omitempty"`
	StoreWhen           string   `yaml:"store_when,omitempty"`
	RecallWhen          string   `yaml:"recall_when,omitempty"`
	RequiresVerifiedFix *bool    `yaml:"requires_verified_fix,omitempty"`
	FailureKind         *bool    `yaml:"failure_kind,omitempty"`
	AutoFingerprint     *bool    `yaml:"auto_fingerprint,omitempty"`
	EnrichName          *bool    `yaml:"enrich_name,omitempty"`
	RecallHints         []string `yaml:"recall_hints,omitempty"`
	StoreHints          []string `yaml:"store_hints,omitempty"`
}

// KindCatalog holds the merged memory kind definitions.
type KindCatalog struct {
	ordered []KindDefinition
	byName  map[string]KindDefinition
}

// LoadKindCatalog merges bundled, global, project, and inline config kinds.
func LoadKindCatalog(cfg *taijicfg.Config) *KindCatalog {
	catalog := &KindCatalog{byName: map[string]KindDefinition{}}
	for _, path := range taijicfg.MemoryKindsSearchPaths(cfg) {
		catalog.mergeFile(path)
	}
	if cfg != nil {
		for _, spec := range cfg.ResolvedMemory().Kinds {
			catalog.mergeSpec(spec)
		}
	}
	if len(catalog.byName) == 0 {
		catalog.mergeBuiltins()
	}
	catalog.rebuildOrder()
	return catalog
}

func (c *KindCatalog) mergeFile(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var file kindFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return
	}
	for _, entry := range file.Kinds {
		c.mergeEntry(entry)
	}
}

func (c *KindCatalog) mergeSpec(spec taijicfg.MemoryKindSpec) {
	c.mergeEntry(kindFileEntry{
		Name:                spec.Name,
		Description:         spec.Description,
		StoreWhen:           spec.StoreWhen,
		RecallWhen:          spec.RecallWhen,
		RequiresVerifiedFix: spec.RequiresVerifiedFix,
		FailureKind:         spec.FailureKind,
		AutoFingerprint:     spec.AutoFingerprint,
		EnrichName:          spec.EnrichName,
		RecallHints:         spec.RecallHints,
		StoreHints:          spec.StoreHints,
	})
}

func (c *KindCatalog) mergeEntry(entry kindFileEntry) {
	name := normalizeKindName(entry.Name)
	if name == "" {
		return
	}
	def := KindDefinition{
		Name:                name,
		Description:         strings.TrimSpace(entry.Description),
		StoreWhen:           strings.TrimSpace(entry.StoreWhen),
		RecallWhen:          strings.TrimSpace(entry.RecallWhen),
		RequiresVerifiedFix: defaultRequiresVerifiedFix(name, entry.RequiresVerifiedFix),
		FailureKind:         boolOrDefault(entry.FailureKind, name == string(KindError) || name == string(KindAntiPattern)),
		AutoFingerprint:     boolOrDefault(entry.AutoFingerprint, name == string(KindError)),
		EnrichName:          boolOrDefault(entry.EnrichName, name == string(KindPreference) || name == string(KindPersona)),
		RecallHints:         normalizeHints(entry.RecallHints),
		StoreHints:          normalizeHints(entry.StoreHints),
	}
	if prev, ok := c.byName[name]; ok {
		def = mergeKindDefinition(prev, def)
	}
	c.byName[name] = def
}

func (c *KindCatalog) mergeBuiltins() {
	for _, entry := range defaultBuiltinKindEntries() {
		c.mergeEntry(entry)
	}
}

func (c *KindCatalog) rebuildOrder() {
	names := make([]string, 0, len(c.byName))
	for name := range c.byName {
		names = append(names, name)
	}
	sort.Strings(names)
	c.ordered = make([]KindDefinition, 0, len(names))
	for _, name := range names {
		c.ordered = append(c.ordered, c.byName[name])
	}
}

// All returns kinds in stable sorted order.
func (c *KindCatalog) All() []KindDefinition {
	if c == nil {
		return nil
	}
	out := make([]KindDefinition, len(c.ordered))
	copy(out, c.ordered)
	return out
}

// Names returns valid kind ids.
func (c *KindCatalog) Names() []string {
	if c == nil {
		return nil
	}
	names := make([]string, len(c.ordered))
	for i, def := range c.ordered {
		names[i] = def.Name
	}
	return names
}

// Lookup returns a kind definition.
func (c *KindCatalog) Lookup(name string) (KindDefinition, bool) {
	if c == nil {
		return KindDefinition{}, false
	}
	def, ok := c.byName[normalizeKindName(name)]
	return def, ok
}

// ParseKind validates raw against the catalog.
func (c *KindCatalog) ParseKind(raw string) (Kind, error) {
	name := normalizeKindName(raw)
	if name == "" {
		return "", errKindRequired
	}
	if c == nil || len(c.byName) == 0 {
		return ParseKind(raw)
	}
	if _, ok := c.byName[name]; !ok {
		return ParseKind(raw)
	}
	return Kind(name), nil
}

// FailureKinds returns kinds used for automatic error recall.
func (c *KindCatalog) FailureKinds() []Kind {
	if c == nil {
		return FailureKinds
	}
	var out []Kind
	for _, def := range c.ordered {
		if def.FailureKind {
			out = append(out, Kind(def.Name))
		}
	}
	if len(out) == 0 {
		return FailureKinds
	}
	return out
}

// ErrorRecallKindNames returns kind filters for tool-error recall (failure + pattern).
func (c *KindCatalog) ErrorRecallKindNames() []string {
	if c == nil {
		return []string{string(KindError), string(KindPattern), string(KindAntiPattern)}
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(name string) {
		name = normalizeKindName(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, def := range c.ordered {
		if def.FailureKind {
			add(def.Name)
		}
	}
	add(string(KindPattern))
	return out
}

// SuggestRecallKinds picks kinds from recall_hints for a user goal.
func (c *KindCatalog) SuggestRecallKinds(goal string) []Kind {
	if c == nil {
		return legacySuggestKinds(strings.ToLower(strings.TrimSpace(goal)))
	}
	goal = strings.ToLower(strings.TrimSpace(goal))
	if goal == "" {
		return nil
	}
	var matched []Kind
	for _, def := range c.ordered {
		if hintsMatch(goal, def.RecallHints) {
			matched = append(matched, Kind(def.Name))
		}
	}
	if len(matched) > 0 {
		return dedupeKinds(matched)
	}
	return legacySuggestKinds(goal)
}

// SuggestStoreKinds picks kinds from store_hints for a user message.
func (c *KindCatalog) SuggestStoreKinds(message string) []Kind {
	if c == nil {
		return nil
	}
	message = strings.ToLower(strings.TrimSpace(message))
	if message == "" {
		return nil
	}
	var matched []Kind
	for _, def := range c.ordered {
		if hintsMatch(message, def.StoreHints) {
			matched = append(matched, Kind(def.Name))
		}
	}
	return dedupeKinds(matched)
}

// RequiresVerifiedFix reports whether a kind needs verified_fix when omitted.
func (c *KindCatalog) RequiresVerifiedFix(kind Kind) bool {
	if c == nil {
		return kind != KindPreference && kind != KindInsight && kind != KindPersona
	}
	if def, ok := c.Lookup(string(kind)); ok {
		return def.RequiresVerifiedFix
	}
	return kind != KindPreference && kind != KindInsight && kind != KindPersona
}

// AutoFingerprint reports whether remember should auto-generate a fingerprint.
func (c *KindCatalog) AutoFingerprint(kind Kind) bool {
	if c == nil {
		return kind == KindError
	}
	if def, ok := c.Lookup(string(kind)); ok {
		return def.AutoFingerprint
	}
	return kind == KindError
}

// EnrichName reports whether name-related recall metadata should be added.
func (c *KindCatalog) EnrichName(kind Kind) bool {
	if c == nil {
		return kind == KindPreference || kind == KindPersona
	}
	if def, ok := c.Lookup(string(kind)); ok {
		return def.EnrichName
	}
	return kind == KindPreference || kind == KindPersona
}

// KindNamesDescription returns a comma-separated list for tool schemas.
func (c *KindCatalog) KindNamesDescription() string {
	names := c.Names()
	if len(names) == 0 {
		return "error, pattern, preference, workflow, insight, anti_pattern, persona"
	}
	return strings.Join(names, ", ")
}

// BuildCatalogAppendix formats the kind catalog for the system prompt.
func (c *KindCatalog) BuildCatalogAppendix() string {
	if c == nil || len(c.ordered) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Memory kinds (catalog)\n\n")
	b.WriteString("Always pass the matching `kind` to memory_recall and memory_remember. Wrong kind is worse than no recall.\n\n")
	b.WriteString("| kind | store when | recall when |\n")
	b.WriteString("|------|------------|-------------|\n")
	for _, def := range c.ordered {
		store := def.StoreWhen
		if store == "" {
			store = def.Description
		}
		recall := def.RecallWhen
		if recall == "" {
			recall = def.Description
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", def.Name, store, recall))
	}
	b.WriteString("\n")
	for _, def := range c.ordered {
		line := "- **" + def.Name + "**"
		if desc := strings.TrimSpace(def.Description); desc != "" {
			line += ": " + desc
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String())
}

func mergeKindDefinition(prev, next KindDefinition) KindDefinition {
	out := next
	if out.Description == "" {
		out.Description = prev.Description
	}
	if out.StoreWhen == "" {
		out.StoreWhen = prev.StoreWhen
	}
	if out.RecallWhen == "" {
		out.RecallWhen = prev.RecallWhen
	}
	if len(out.RecallHints) == 0 {
		out.RecallHints = prev.RecallHints
	} else {
		out.RecallHints = dedupeStrings(append(prev.RecallHints, out.RecallHints...))
	}
	if len(out.StoreHints) == 0 {
		out.StoreHints = prev.StoreHints
	} else {
		out.StoreHints = dedupeStrings(append(prev.StoreHints, out.StoreHints...))
	}
	return out
}

func defaultRequiresVerifiedFix(name string, override *bool) bool {
	if override != nil {
		return *override
	}
	switch name {
	case string(KindPreference), string(KindInsight), string(KindPersona):
		return false
	default:
		return true
	}
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value != nil {
		return *value
	}
	return fallback
}

func normalizeKindName(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}

func normalizeHints(values []string) []string {
	return dedupeStrings(values)
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func hintsMatch(goal string, hints []string) bool {
	for _, hint := range hints {
		if hint != "" && strings.Contains(goal, hint) {
			return true
		}
	}
	return false
}

func dedupeKinds(values []Kind) []Kind {
	seen := map[Kind]struct{}{}
	out := make([]Kind, 0, len(values))
	for _, kind := range values {
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		out = append(out, kind)
	}
	return out
}

func legacySuggestKinds(goal string) []Kind {
	recallHints := []string{
		"erinner", "remember", "gedächtnis", "gedaechtnis", "memory tool",
		"memory_recall", "merk dir", "hast du dir gemerkt", "what do you know about me",
	}
	for _, hint := range recallHints {
		if strings.Contains(goal, hint) {
			return []Kind{KindPreference, KindPersona, KindPattern, KindInsight}
		}
	}
	return nil
}

func defaultBuiltinKindEntries() []kindFileEntry {
	path := taijicfg.BundledMemoryKindsPath("")
	if path != "" {
		raw, err := os.ReadFile(path)
		if err == nil {
			var file kindFile
			if yaml.Unmarshal(raw, &file) == nil && len(file.Kinds) > 0 {
				return file.Kinds
			}
		}
	}
	return []kindFileEntry{
		{Name: "error", Description: "Verified fix for a tool or shell failure", RequiresVerifiedFix: boolPtr(true), FailureKind: boolPtr(true), AutoFingerprint: boolPtr(true)},
		{Name: "anti_pattern", Description: "An approach that did not work", RequiresVerifiedFix: boolPtr(true), FailureKind: boolPtr(true)},
		{Name: "pattern", Description: "Repository or project convention", RequiresVerifiedFix: boolPtr(true)},
		{Name: "workflow", Description: "Multi-step process that worked", RequiresVerifiedFix: boolPtr(true)},
		{Name: "preference", Description: "Stable user preference or personal fact", RequiresVerifiedFix: boolPtr(false), EnrichName: boolPtr(true)},
		{Name: "persona", Description: "Assistant identity — name, role, tone", RequiresVerifiedFix: boolPtr(false), EnrichName: boolPtr(true)},
		{Name: "insight", Description: "Non-obvious architecture or setup note", RequiresVerifiedFix: boolPtr(false)},
	}
}

func boolPtr(v bool) *bool {
	return &v
}
