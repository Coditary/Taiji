package config

import (
	"os"
	"path/filepath"
)

const packsSubdir = "packs"

// PacksConfig controls context pack folders and assembly.
type PacksConfig struct {
	Enabled        *bool    `yaml:"enabled,omitempty"`
	Dirs           []string `yaml:"dirs,omitempty"`
	MaxChars       int      `yaml:"max_chars,omitempty"`
	KeywordBM25    *bool    `yaml:"keyword_bm25,omitempty"`
	Semantic       *bool    `yaml:"semantic,omitempty"`
	SemanticTopK   int      `yaml:"semantic_top_k,omitempty"`
	MinBM25Score   float32  `yaml:"min_bm25_score,omitempty"`
	MinSemanticScore float32 `yaml:"min_semantic_score,omitempty"`
	RAGCollection  string   `yaml:"rag_collection,omitempty"`
}

func defaultPacksConfig() PacksConfig {
	enabled := true
	keywordBM25 := true
	semantic := true
	return PacksConfig{
		Enabled:     &enabled,
		KeywordBM25: &keywordBM25,
		Semantic:    &semantic,
	}
}

// ResolvedPacks returns packs config with defaults.
func (c *Config) ResolvedPacks() PacksConfig {
	if c == nil {
		return defaultPacksConfig()
	}
	out := c.Packs
	if out.Enabled == nil {
		def := defaultPacksConfig()
		out.Enabled = def.Enabled
	}
	if out.KeywordBM25 == nil {
		out.KeywordBM25 = defaultPacksConfig().KeywordBM25
	}
	if out.Semantic == nil {
		out.Semantic = defaultPacksConfig().Semantic
	}
	return out
}

func (p PacksConfig) EnabledOrDefault() bool {
	if p.Enabled == nil {
		return true
	}
	return *p.Enabled
}

func (p PacksConfig) MaxCharsOrDefault() int {
	if p.MaxChars <= 0 {
		return 16000
	}
	return p.MaxChars
}

func (p PacksConfig) KeywordBM25OrDefault() bool {
	if p.KeywordBM25 == nil {
		return true
	}
	return *p.KeywordBM25
}

func (p PacksConfig) SemanticOrDefault() bool {
	if p.Semantic == nil {
		return true
	}
	return *p.Semantic
}

func (p PacksConfig) SemanticTopKOrDefault() int {
	if p.SemanticTopK <= 0 {
		return 4
	}
	return p.SemanticTopK
}

func (p PacksConfig) MinBM25ScoreOrDefault() float32 {
	if p.MinBM25Score <= 0 {
		return 0.45
	}
	return p.MinBM25Score
}

func (p PacksConfig) MinSemanticScoreOrDefault() float32 {
	if p.MinSemanticScore <= 0 {
		return 0.55
	}
	return p.MinSemanticScore
}

// GlobalPacksDir returns ~/.taiji/packs (or XDG).
func GlobalPacksDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "taiji", packsSubdir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "taiji", packsSubdir)
}

// ProjectPacksDir returns the project packs directory (managed or local runtime).
func ProjectPacksDir(root string) string {
	return filepath.Join(RuntimeDir(root), packsSubdir)
}

// DefaultPacksDirs returns pack search paths (project, global).
func DefaultPacksDirs(root string) []string {
	var out []string
	if dir := ProjectPacksDir(root); dir != "" {
		out = append(out, dir)
	}
	if dir := GlobalPacksDir(); dir != "" {
		out = append(out, dir)
	}
	return out
}

func ensurePacksDirs(cfg *Config) {
	if cfg == nil || len(cfg.Packs.Dirs) > 0 {
		return
	}
	cfg.Packs.Dirs = DefaultPacksDirs(cfg.Root)
}
