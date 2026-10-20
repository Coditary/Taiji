package builtin

import (
	"sync"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

var workspaceCfg struct {
	mu           sync.RWMutex
	sessionsRoot string
	projectRoot  string
	rag          *rag.Service
}

// ConfigureWorkspace sets session storage roots and optional RAG for checkpoint restore indexing.
func ConfigureWorkspace(cfg *taijicfg.Config, ragSvc *rag.Service) {
	workspaceCfg.mu.Lock()
	defer workspaceCfg.mu.Unlock()
	workspaceCfg.sessionsRoot = ""
	workspaceCfg.projectRoot = "."
	if cfg != nil {
		if root := cfg.Root; root != "" {
			workspaceCfg.projectRoot = root
			workspaceCfg.sessionsRoot = taijicfg.SessionsDir(root)
		}
	}
	workspaceCfg.rag = ragSvc
}

// SetWorkspaceRootsForTest overrides project and session roots (tests only).
func SetWorkspaceRootsForTest(projectRoot, sessionsRoot string) {
	workspaceCfg.mu.Lock()
	workspaceCfg.projectRoot = projectRoot
	workspaceCfg.sessionsRoot = sessionsRoot
	workspaceCfg.mu.Unlock()
}

func workspaceSessionsRoot() string {
	workspaceCfg.mu.RLock()
	root := workspaceCfg.sessionsRoot
	workspaceCfg.mu.RUnlock()
	if root == "" {
		return ""
	}
	return root
}

func workspaceProjectRoot() string {
	workspaceCfg.mu.RLock()
	root := workspaceCfg.projectRoot
	workspaceCfg.mu.RUnlock()
	if root == "" {
		return "."
	}
	return root
}

func workspaceRAG() *rag.Service {
	workspaceCfg.mu.RLock()
	svc := workspaceCfg.rag
	workspaceCfg.mu.RUnlock()
	return svc
}
