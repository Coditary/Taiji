package memory_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/tools"
)

func TestIntegrationMemoryFlow(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:   root,
		Memory: taijicfg.DefaultMemoryConfig(),
	}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")

	if err := os.MkdirAll(taijicfg.ProjectMemoryDir(root), 0o755); err != nil {
		t.Fatal(err)
	}

	svc := memory.NewService(cfg, nil)
	if svc == nil || !svc.Enabled() {
		t.Fatal("memory service should be enabled")
	}

	ctx := context.Background()
	saved, err := svc.Remember(ctx, memory.RememberRequest{
		Kind:        string(memory.KindError),
		Title:       "Test npm ENOENT",
		RecallText:  "npm install fails with ENOENT for missing package",
		VerifiedFix: "Run npm install missing-package",
		Fingerprint: "npm:enoent:*",
		Scope:       "local",
	})
	if err != nil {
		t.Fatalf("remember: %v", err)
	}
	indexPath := filepath.Join(taijicfg.ProjectMemoryDir(root), "entries.json")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("memory index missing: %v", err)
	}

	entries, err := svc.Recall(ctx, memory.RecallRequest{
		Query:       "npm ERR! enoent some-package",
		Kind:        string(memory.KindError),
		Fingerprint: "npm ERR! enoent some-package",
		TopK:        3,
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one recall hit")
	}
	if entries[0].MatchSource != "exact" {
		t.Fatalf("expected exact match, got %q", entries[0].MatchSource)
	}

	hint := svc.FormatErrorHint(ctx, "shell", "error: npm ERR! enoent some-package")
	if hint == "" || !strings.Contains(hint, "Test npm ENOENT") {
		t.Fatalf("unexpected error hint: %q", hint)
	}

	revised, err := svc.Revise(ctx, memory.ReviseRequest{
		ID:           saved.ID,
		RevisionNote: "Added scope note after integration test",
		Outcome:      "success",
		VerifiedFix:  "Run npm install missing-package --save",
	})
	if err != nil {
		t.Fatalf("revise: %v", err)
	}
	if revised.Hits != 1 {
		t.Fatalf("expected hits=1, got %d", revised.Hits)
	}
	loaded, err := memory.LoadFile(revised.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(loaded.RevisionLog, "Added scope note") {
		t.Fatalf("revision not persisted: %s", loaded.RevisionLog)
	}
}

func TestIntegrationMemoryToolsViaRuntime(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:   root,
		Memory: taijicfg.DefaultMemoryConfig(),
	}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	disabled := false
	cfg.RAG.Enabled = &disabled // RAG optional for memory tools

	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Local: true})
	if err != nil {
		t.Fatalf("runtime: %v", err)
	}
	defer rt.Close()

	if rt.Memory == nil || !rt.Memory.Enabled() {
		t.Fatal("runtime memory unavailable")
	}

	reg := rt.Tools
	ctx := context.Background()

	rememberArgs, _ := json.Marshal(map[string]any{
		"kind":         "pattern",
		"title":        "Run tests with make",
		"recall_text":  "how to run project tests",
		"verified_fix": "make -C taiji-ai test",
		"scope":        "local",
	})
	rememberRes, err := reg.Run(ctx, "memory_remember", string(rememberArgs))
	if err != nil {
		t.Fatalf("memory_remember tool: %v", err)
	}
	rememberOut := rememberRes.Text
	if !strings.Contains(rememberOut, "remembered id=") {
		t.Fatalf("unexpected remember output: %q", rememberOut)
	}

	recallArgs, _ := json.Marshal(map[string]any{
		"query": "how to run project tests",
		"kind":  "pattern",
	})
	recallRes, err := reg.Run(ctx, "memory_recall", string(recallArgs))
	if err != nil {
		t.Fatalf("memory_recall tool: %v", err)
	}
	recallOut := recallRes.Text
	if !strings.Contains(recallOut, "Run tests with make") {
		t.Fatalf("unexpected recall output: %q", recallOut)
	}

	id := ""
	for _, part := range strings.Fields(rememberOut) {
		if strings.HasPrefix(part, "id=") {
			id = strings.TrimPrefix(part, "id=")
			break
		}
	}
	if id == "" {
		t.Fatalf("could not parse id from %q", rememberOut)
	}

	reviseArgs, _ := json.Marshal(map[string]any{
		"id":            id,
		"revision_note": "tool integration test update",
		"outcome":       "hit",
		"verified_fix":  "make -C taiji-ai test ./...",
	})
	reviseRes, err := reg.Run(ctx, "memory_revise", string(reviseArgs))
	if err != nil {
		t.Fatalf("memory_revise tool: %v", err)
	}
	reviseOut := reviseRes.Text
	if !strings.Contains(reviseOut, "revised id=") {
		t.Fatalf("unexpected revise output: %q", reviseOut)
	}

	names := map[string]bool{}
	for _, def := range reg.Definitions() {
		names[def.Name] = true
	}
	for _, want := range []string{"memory_recall", "memory_remember", "memory_revise"} {
		if !names[want] {
			t.Fatalf("tool %q not registered", want)
		}
	}
	_ = tools.Definition{}
}
