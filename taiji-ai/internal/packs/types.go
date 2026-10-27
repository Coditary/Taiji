package packs

// InjectionMode controls how a pack node enters the prompt.
type InjectionMode string

const (
	ModeAlways   InjectionMode = "always"
	ModeKeyword  InjectionMode = "keyword"
	ModeSemantic InjectionMode = "semantic"
	ModeManual   InjectionMode = "manual"
)

// NodeFrontmatter is YAML metadata for a pack markdown file.
type NodeFrontmatter struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Injection   InjectionMode `yaml:"injection"`
	Keywords    []string      `yaml:"keywords"`
	When        string        `yaml:"when"`
	Priority    int           `yaml:"priority"`
	// Legacy SKILL.md
	DisableModelInvocation bool `yaml:"disable-model-invocation"`
	DisableModelInvocationSnake bool `yaml:"disable_model_invocation"`
}

// PackNode is one markdown document in a context pack.
type PackNode struct {
	PackName string
	RelPath  string
	Path     string
	Body     string
	Meta     NodeFrontmatter
	Legacy   LegacyKind
}

// LegacyKind marks nodes synthesized from pre-pack subsystems.
type LegacyKind string

const (
	LegacyNone      LegacyKind = ""
	LegacyAgentsMD  LegacyKind = "agents"
	LegacySkill     LegacyKind = "skill"
	LegacyDecision  LegacyKind = "decision"
)

// Pack is a named folder of nodes (.taiji/packs/<name>/…).
type Pack struct {
	Name  string
	Dir   string
	Nodes []PackNode
}

// NodeID returns a stable id for mentions (pack:relative/path).
func (n PackNode) NodeID() string {
	if n.Legacy == LegacyAgentsMD {
		return "agents"
	}
	if n.PackName == "" {
		return n.RelPath
	}
	return n.PackName + "/" + n.RelPath
}

// DisplayName returns the frontmatter name or file stem.
func (n PackNode) DisplayName() string {
	if name := trim(n.Meta.Name); name != "" {
		return name
	}
	return n.RelPath
}

// EffectiveInjection resolves default injection modes (legacy-compatible).
func (n PackNode) EffectiveInjection() InjectionMode {
	if mode := normalizeInjection(n.Meta.Injection); mode != "" {
		return mode
	}
	switch n.Legacy {
	case LegacyAgentsMD:
		return ModeAlways
	case LegacySkill, LegacyDecision:
		return ModeManual
	default:
		return ModeKeyword
	}
}

func normalizeInjection(m InjectionMode) InjectionMode {
	switch trim(string(m)) {
	case "always":
		return ModeAlways
	case "keyword":
		return ModeKeyword
	case "semantic":
		return ModeSemantic
	case "manual":
		return ModeManual
	default:
		return ""
	}
}

func trim(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}
