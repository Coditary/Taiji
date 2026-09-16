package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
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
	Skills    *skills.Loader
	MCP       *mcp.Bridge
	RAG       *rag.Service
}

// New builds runtime services from config.
func New(ctx context.Context, cfg *taijicfg.Config, opts Options) (*Runtime, error) {
	reg, err := provider.NewRegistry(ctx, cfg)
	if err != nil {
		return nil, err
	}
	toolReg := tools.NewRegistry()
	builtin.Register(toolReg)

	ragSvc, err := rag.NewService(cfg, rag.Options{Local: opts.Local, LocalOnly: opts.LocalOnly})
	if err != nil {
		fmt.Fprintf(os.Stderr, "taiji: rag unavailable: %v\n", err)
	} else {
		builtin.EnableRAGOrchestration(toolReg, ragSvc)
		fmt.Fprintf(os.Stderr, "taiji: rag scope: %s\n", ragSvc.DescribeScope())
	}

	bridge, err := mcp.AttachFromConfig(ctx, cfg, toolReg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "taiji: mcp unavailable: %v\n", err)
		bridge = mcp.NewBridge()
	} else if len(cfg.MCP.Servers) > 0 {
		fmt.Fprintf(os.Stderr, "taiji: mcp connected: %s\n", strings.Join(bridge.ConnectedServers(), ", "))
	}
	return &Runtime{
		Config:    cfg,
		Providers: reg,
		Tools:     toolReg,
		Skills:    skills.NewLoader(cfg.Skills.Dirs),
		MCP:       bridge,
		RAG:       ragSvc,
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
	if r.MCP != nil {
		if closeErr := r.MCP.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}
