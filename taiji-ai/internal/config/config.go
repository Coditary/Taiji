package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const dirName = ".taiji"
const configFile = "config.yaml"

// Config is the Taiji project configuration.
type Config struct {
	DefaultProvider string                    `yaml:"default_provider"`
	Wuji            WujiConfig                `yaml:"wuji"`
	Agent           AgentConfig               `yaml:"agent"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
	RAG             RAGConfig                 `yaml:"rag"`
	Decisions       DecisionConfig            `yaml:"decisions"`
	Memory          MemoryConfig              `yaml:"memory"`
	MCP             MCPConfig                 `yaml:"mcp"`
	Skills          SkillsConfig              `yaml:"skills"`
	Packs           PacksConfig               `yaml:"packs"`
	Hooks           HooksConfig               `yaml:"hooks"`
	UI              UIConfig                  `yaml:"ui"`
	STT             STTConfig                 `yaml:"stt"`
	Tools           ToolsConfig               `yaml:"tools"`
	Root            string                    `yaml:"-"`
	// StorageLocal is true when runtime data lives in <root>/.taiji (e.g. --local).
	StorageLocal bool `yaml:"-"`
}

// WujiConfig controls provider refresh from a Wuji core daemon.
type WujiConfig struct {
	// AutoSync refreshes the provider catalog from Wuji on each command (and saves when changed).
	// When false, use: taiji providers refresh
	AutoSync bool   `yaml:"auto_sync"`
	Endpoint string `yaml:"endpoint"`
	WujiRoot string `yaml:"wuji_root"`
}

// AgentConfig holds agent loop defaults.
type AgentConfig struct {
	MaxSteps         int                `yaml:"max_steps"`
	SystemPrompt     string             `yaml:"system_prompt"`
	ChainOfThought   ChainOfThoughtMode `yaml:"chain_of_thought"`
	ThoughtMaxChars  int                `yaml:"thought_max_chars"`
	ContinueTrigger  string             `yaml:"continue_trigger"`
	MaxContinues     int                `yaml:"max_continues"`
	ContinuePrompt   string             `yaml:"continue_prompt"`
	AutoNameSessions     *bool `yaml:"auto_name_sessions,omitempty"`
	DefaultContextWindow      int   `yaml:"default_context_window,omitempty"`
	PruneStaleObservations    *bool `yaml:"prune_stale_observations,omitempty"`
	ThrashingThreshold        int   `yaml:"thrashing_threshold,omitempty"`
	CacheFriendlyPruning      *bool `yaml:"cache_friendly_pruning,omitempty"`
	MaxContextChars           *int  `yaml:"max_context_chars,omitempty"`
	DelegateMaxConcurrent     int   `yaml:"delegate_max_concurrent,omitempty"`
}

// AutoNameSessionsEnabled reports whether chat sessions should get an LLM-generated title.
func (a AgentConfig) AutoNameSessionsEnabled() bool {
	if a.AutoNameSessions == nil {
		return true
	}
	return *a.AutoNameSessions
}

// ProviderConfig is a YAML-defined provider. The `type` field selects the adapter.
type ProviderConfig struct {
	Type string `yaml:"type"`

	// Wuji backend — Taiji only needs provider_id + optional model override.
	Endpoint   string `yaml:"endpoint"`
	WujiRoot   string `yaml:"wuji_root"`
	ProviderID string `yaml:"provider_id"` // id from .wuji/config.yaml providers
	Model      string `yaml:"model"`

	// OpenAI-compatible
	BaseURL   string `yaml:"base_url"`
	APIKey    string `yaml:"api_key"`
	APIKeyEnv string `yaml:"api_key_env"`

	// Anthropic
	AnthropicVersion string `yaml:"anthropic_version"`

	// Shared generation defaults
	MaxTokens      int     `yaml:"max_tokens"`
	ContextWindow  int     `yaml:"context_window,omitempty"`
	Temperature    float64 `yaml:"temperature"`

	// Synced is set when imported from Wuji (refresh may update model/endpoint).
	Synced bool `yaml:"synced,omitempty"`
	// Pinned prevents refresh from overwriting this entry.
	Pinned bool `yaml:"pinned,omitempty"`

	// WujiKind is set by taiji providers refresh: driver | api | catalog | provider.
	WujiKind string `yaml:"wuji_kind,omitempty"`
	// DisplayName is a human label from Wuji (catalog name, etc.).
	DisplayName string `yaml:"display_name,omitempty"`
}

// MCPConfig references MCP server definitions.
type MCPConfig struct {
	ConfigPath string   `yaml:"config"`
	Servers    []string `yaml:"servers"`
}

// SkillsConfig lists directories containing SKILL.md files.
type SkillsConfig struct {
	Dirs []string        `yaml:"dirs"`
	RAG  SkillsRAGConfig `yaml:"rag"`
}

// LoadOptions controls how the Taiji project root is resolved.
type LoadOptions struct {
	// Local uses the current working directory as root instead of walking up
	// to a parent go.mod or .taiji/config.yaml.
	Local bool
}

// LoadAt reads project config from the managed store or <root>/.taiji when local.
func LoadAt(root string, opts LoadOptions) (*Config, error) {
	return loadAtRoot(root, opts)
}

func loadAtRoot(root string, opts LoadOptions) (*Config, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("config root is not set")
	}
	storageLocal := ResolveStorageLocal(root, opts)
	path := resolveProjectConfigPath(root, storageLocal)
	cfg := defaultConfig(root)
	cfg.StorageLocal = storageLocal
	if _, err := os.Stat(path); os.IsNotExist(err) {
		applyGlobalMCPDefaults(cfg)
		if !storageLocal {
			applyGlobalUserPrefs(cfg)
		}
		ensureDecisionProfileDirs(cfg)
		ensureMemoryDirs(cfg)
		normalizeConfig(cfg)
		return cfg, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.Root = root
	cfg.StorageLocal = storageLocal
	sanitizeLegacyWujiPaths(cfg)
	applyGlobalMCPDefaults(cfg)
	if !storageLocal {
		applyGlobalUserPrefs(cfg)
	}
	ensureSkillsDirs(cfg)
	ensurePacksDirs(cfg)
	ensureDecisionProfileDirs(cfg)
	ensureMemoryDirs(cfg)
	normalizeConfig(cfg)
	return cfg, nil
}

// Load reads .taiji/config.yaml from the project root.
func Load(opts LoadOptions) (*Config, error) {
	root, err := Root(opts)
	if err != nil {
		return nil, err
	}
	return loadAtRoot(root, opts)
}

func defaultConfig(root string) *Config {
	return &Config{
		Root:            root,
		DefaultProvider: "",
		Wuji: WujiConfig{
			AutoSync: false,
			Endpoint: "auto",
			WujiRoot: "",
		},
		Agent: AgentConfig{
			MaxSteps:       25,
			ChainOfThought: ChainOfThoughtOff,
		},
		Providers: map[string]ProviderConfig{},
		RAG:       DefaultRAGConfig(),
		Decisions: func() DecisionConfig {
			d := DefaultDecisionConfig()
			d.Dirs = DefaultDecisionProfileDirs(root)
			return d
		}(),
		Memory: DefaultMemoryConfig(),
		Tools:  DefaultToolsConfig(),
		MCP: MCPConfig{
			ConfigPath: filepath.Join(root, ".wuji", "mcp.json"),
		},
		Skills: SkillsConfig{
			Dirs: DefaultSkillsDirs(root),
		},
		Packs: defaultPacksConfig(),
		UI:  DefaultUIConfig(),
		STT: DefaultSTTConfig(),
	}
}

func normalizeConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	cfg.UI.TurnMetrics.normalize()
}

// Root returns the Taiji project root. Search order:
// 1) TAIJI_ROOT env
// 2) cwd when opts.Local is true
// 3) .taiji/config.yaml or go.mod walking up from cwd
// 4) same walk starting from the taiji binary directory
// 5) cwd fallback
func Root(opts LoadOptions) (string, error) {
	if v := os.Getenv("TAIJI_ROOT"); v != "" {
		return v, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if opts.Local {
		abs, err := filepath.Abs(wd)
		if err != nil {
			return wd, nil
		}
		return abs, nil
	}
	if root, ok := findProjectRoot(wd); ok {
		return root, nil
	}
	if exe, err := os.Executable(); err == nil {
		if root, ok := findProjectRoot(filepath.Dir(exe)); ok {
			return root, nil
		}
	}
	return wd, nil
}

func findTaijiConfigRoot(start string) (string, bool) {
	dir := start
	for {
		if hasTaijiConfig(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func findProjectRoot(start string) (string, bool) {
	dir := start
	for {
		if hasTaijiConfig(dir) || hasGoMod(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func hasTaijiConfig(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, dirName, configFile))
	return err == nil
}

func hasGoMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

// RuntimeDir returns the Taiji runtime directory for a project (managed or local).
func RuntimeDir(root string) string {
	return RuntimeDirForRoot(root, ProjectStorageLocal(root))
}

// SessionsDir returns the sessions directory for a project.
func SessionsDir(root string) string {
	return filepath.Join(RuntimeDir(root), "sessions")
}

type fileConfig struct {
	DefaultProvider string                    `yaml:"default_provider,omitempty"`
	Wuji            WujiConfig                `yaml:"wuji,omitempty"`
	Agent           AgentConfig               `yaml:"agent,omitempty"`
	Providers       map[string]ProviderConfig `yaml:"providers,omitempty"`
	RAG             RAGConfig                 `yaml:"rag,omitempty"`
	Decisions       DecisionConfig            `yaml:"decisions,omitempty"`
	Memory          MemoryConfig              `yaml:"memory,omitempty"`
	MCP             MCPConfig                 `yaml:"mcp,omitempty"`
	Skills          SkillsConfig              `yaml:"skills,omitempty"`
	Packs           PacksConfig               `yaml:"packs,omitempty"`
	Hooks           HooksConfig               `yaml:"hooks,omitempty"`
	UI              UIConfig                  `yaml:"ui,omitempty"`
	STT             STTConfig                 `yaml:"stt,omitempty"`
	Tools           ToolsConfig               `yaml:"tools,omitempty"`
}

// Save writes project config to the managed store or <root>/.taiji when local.
func (c *Config) Save() error {
	if c == nil || c.Root == "" {
		return fmt.Errorf("config root is not set")
	}
	dir := RuntimeDirForRoot(c.Root, c.StorageLocal)
	if !c.StorageLocal {
		if err := ensureManagedProjectRootMarker(c.Root); err != nil {
			return fmt.Errorf("project marker: %w", err)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	fc := fileConfig{
		DefaultProvider: c.DefaultProvider,
		Wuji:            c.Wuji,
		Agent:           c.Agent,
		Providers:       c.Providers,
		RAG:             c.RAG,
		Decisions:       c.Decisions,
		Memory:          c.Memory,
		MCP:             c.MCP,
		Skills:          c.Skills,
		Packs:           c.Packs,
		Hooks:           c.Hooks,
		Tools:           c.Tools,
	}
	if c.StorageLocal {
		fc.UI = c.UI
		fc.STT = c.STT
	}
	if err := SaveGlobalUserPrefs(c.UI, c.STT); err != nil {
		return err
	}
	raw, err := yaml.Marshal(fc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path := filepath.Join(dir, configFile)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
