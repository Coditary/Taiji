package rag

import (
	"path/filepath"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// ScopeMode controls where RAG data is read and written.
type ScopeMode string

const (
	// ScopeGlobal reads and writes the shared Wuji project store.
	ScopeGlobal ScopeMode = "global"
	// ScopeLocal writes to the Taiji project root; reads local + global collections.
	ScopeLocal ScopeMode = "local"
	// ScopeLocalOnly writes and reads only the Taiji project root store.
	ScopeLocalOnly ScopeMode = "local-only"

	wujiStoreDir  = ".wuji"
	taijiStoreDir = ".taiji"
)

// Scope resolves store roots for local vs global RAG.
type Scope struct {
	Mode       ScopeMode
	LocalRoot  string
	GlobalRoot string
}

// ScopeOptions mirrors Taiji CLI flags.
type ScopeOptions struct {
	Local     bool
	LocalOnly bool
}

// ScopeFromConfig builds a scope from config and CLI flags.
func ScopeFromConfig(cfg *taijicfg.Config, opts ScopeOptions) Scope {
	localRoot := "."
	globalRoot := "."
	if cfg != nil {
		if cfg.Root != "" {
			localRoot = cfg.Root
		}
		globalRoot = cfg.ResolvedRAG().GlobalStoreRoot(localRoot, cfg.Wuji.WujiRoot)
	}
	mode := ScopeGlobal
	if opts.LocalOnly {
		mode = ScopeLocalOnly
	} else if opts.Local {
		mode = ScopeLocal
	}
	return Scope{
		Mode:       mode,
		LocalRoot:  filepath.Clean(localRoot),
		GlobalRoot: filepath.Clean(globalRoot),
	}
}

func (s Scope) WriteRoot() string {
	switch s.Mode {
	case ScopeLocal, ScopeLocalOnly:
		return s.LocalRoot
	default:
		return s.GlobalRoot
	}
}

func (s Scope) ReadRoots() []scopedRoot {
	switch s.Mode {
	case ScopeLocalOnly:
		return []scopedRoot{{Label: "local", Root: s.LocalRoot, StoreDir: taijiStoreDir}}
	case ScopeLocal:
		return []scopedRoot{
			{Label: "local", Root: s.LocalRoot, StoreDir: taijiStoreDir},
			{Label: "global", Root: s.GlobalRoot, StoreDir: wujiStoreDir},
		}
	default:
		return []scopedRoot{{Label: "global", Root: s.GlobalRoot, StoreDir: wujiStoreDir}}
	}
}

type scopedRoot struct {
	Label    string
	Root     string
	StoreDir string
}

// WriteStoreDir returns the state directory for local RAG writes (.taiji) or global (.wuji).
func (s Scope) WriteStoreDir() string {
	switch s.Mode {
	case ScopeLocal, ScopeLocalOnly:
		return taijiStoreDir
	default:
		return wujiStoreDir
	}
}

func localStorePath(root string) string {
	return filepath.Join(root, taijiStoreDir)
}

func globalStorePath(root string) string {
	return filepath.Join(root, wujiStoreDir)
}

// Describe returns a short human-readable scope summary for tools and logs.
func (s Scope) Describe() string {
	switch s.Mode {
	case ScopeLocalOnly:
		return "local-only (read/write " + localStorePath(s.LocalRoot) + ")"
	case ScopeLocal:
		return "local (write " + localStorePath(s.LocalRoot) + ", read local+global)"
	default:
		return "global (read/write " + globalStorePath(s.GlobalRoot) + ")"
	}
}

// SamePhysicalStore reports whether local and global roots are identical.
func (s Scope) SamePhysicalStore() bool {
	a, errA := filepath.Abs(s.LocalRoot)
	b, errB := filepath.Abs(s.GlobalRoot)
	if errA != nil || errB != nil {
		return strings.EqualFold(filepath.Clean(s.LocalRoot), filepath.Clean(s.GlobalRoot))
	}
	return a == b
}
