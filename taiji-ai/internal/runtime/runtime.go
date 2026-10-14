package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/decision"
	"github.com/coditary/taiji/internal/decisions"
	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/question"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
	"github.com/coditary/taiji/internal/tools/mcp"
)

// Options configures runtime construction.
type Options struct {
	Local     bool
	LocalOnly bool
}

// Runtime holds shared Taiji services.
type Runtime struct {
	Config    *taijicfg.Config
	Providers *provider.Registry
	Tools     *tools.Registry
	Skills        *skills.Loader
	SkillsCatalog *skills.Catalog
	MCP           *mcp.Bridge
	RAG       *rag.Service
	Decisions         *decision.Service
	DecisionProfiles  *decisions.Loader
	Memory            *memory.Service
	Questions         *question.Service
	Hooks             *hooks.Runner
}

// New builds runtime services from config.
func New(ctx context.Context, cfg *taijicfg.Config, opts Options) (*Runtime, error) {
	if cfg != nil {
		root := provider.ResolveWujiRoot(cfg)
		endpoint := provider.ResolveTaijiWujiEndpoint(cfg.Root, cfg.Wuji.Endpoint, cfg.Wuji.WujiRoot)
		if err := provider.EnsureWujiCore(ctx, root, endpoint); err != nil {
			fmt.Fprintf(os.Stderr, "taiji: wuji core unavailable: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "taiji: wuji core: %s\n", endpoint)
		}
	}
	reg, err := provider.NewRegistry(ctx, cfg)
	if err != nil {
		return nil, err
	}
	toolReg := tools.NewRegistry()
	builtin.Register(toolReg)
	skillsLoader := skills.NewLoader(cfg.Skills.Dirs)
	builtin.EnableSkillOrchestration(toolReg, skillsLoader)

	ragSvc, err := rag.NewService(cfg, rag.Options{Local: opts.Local, LocalOnly: opts.LocalOnly})
	if err != nil {
		fmt.Fprintf(os.Stderr, "taiji: rag unavailable: %v\n", err)
	} else {
		builtin.EnableRAGOrchestration(toolReg, ragSvc)
		fmt.Fprintf(os.Stderr, "taiji: rag scope: %s\n", ragSvc.DescribeScope())
	}
	skillsCatalog := skills.NewCatalog(skillsLoader, ragSvc, cfg)
	if skillsCatalog != nil && skillsCatalog.Enabled() {
		if err := skillsCatalog.EnsureIndexed(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "taiji: skills index: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "taiji: skills search: collection=%s\n", skillsCatalog.Collection())
		}
	}
	builtin.EnableImageGeneration(toolReg, cfg)

	var profileLoader *decisions.Loader
	decSvc, err := decision.NewService(cfg)
	if err != nil {
		if err != decision.ErrDisabled {
			fmt.Fprintf(os.Stderr, "taiji: decisions unavailable: %v\n", err)
		}
	} else {
		profileLoader = decisions.NewLoader(cfg.ResolvedDecision().Dirs)
		builtin.EnableDecisionOrchestration(toolReg, decSvc, profileLoader, taijicfg.ProjectDecisionsDir(cfg.Root))
		fmt.Fprintf(os.Stderr, "taiji: decisions: %s\n", decSvc.DescribeBackend())
		if profiles, err := profileLoader.LoadAll(); err == nil && len(profiles) > 0 {
			fmt.Fprintf(os.Stderr, "taiji: decision profiles: %d loaded\n", len(profiles))
		}
	}

	memSvc := memory.NewService(cfg, ragSvc)
	builtin.EnableMemoryOrchestration(toolReg, memSvc)
	if memSvc != nil && memSvc.Enabled() {
		fmt.Fprintf(os.Stderr, "taiji: memory: %s\n", memSvc.DescribeDirs())
	}

	questionsSvc := question.NewService(question.RejectPrompter{})
	builtin.EnableAskUser(toolReg, questionsSvc)

	bridge, err := mcp.AttachFromConfig(ctx, cfg, toolReg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "taiji: mcp unavailable: %v\n", err)
		bridge = mcp.NewBridge()
	} else if len(cfg.MCP.Servers) > 0 {
		fmt.Fprintf(os.Stderr, "taiji: mcp connected: %s\n", strings.Join(bridge.ConnectedServers(), ", "))
	}

	var hookRunner *hooks.Runner
	if cfg != nil && cfg.Hooks.HooksEnabled() {
		if loaded, err := hooks.LoadFromPaths(taijicfg.HookSearchPaths(cfg)); err != nil {
			fmt.Fprintf(os.Stderr, "taiji: hooks unavailable: %v\n", err)
		} else {
			hookRunner = loaded
			if hookRunner != nil && hookRunner.Enabled() {
				fmt.Fprintf(os.Stderr, "taiji: hooks loaded\n")
			}
		}
	}

	return &Runtime{
		Config:    cfg,
		Providers: reg,
		Tools:     toolReg,
		Skills:        skillsLoader,
		SkillsCatalog: skillsCatalog,
		MCP:           bridge,
		RAG:       ragSvc,
		Decisions:        decSvc,
		DecisionProfiles: profileLoader,
		Memory:           memSvc,
		Questions:        questionsSvc,
		Hooks:            hookRunner,
	}, nil
}

// Close releases runtime resources.
func (r *Runtime) Close() error {
	var err error
	if r.RAG != nil {
		if closeErr := r.RAG.Close(); closeErr != nil {
			err = closeErr
		}
	}
	if r.Decisions != nil {
		if closeErr := r.Decisions.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	if r.MCP != nil {
		if closeErr := r.MCP.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}
