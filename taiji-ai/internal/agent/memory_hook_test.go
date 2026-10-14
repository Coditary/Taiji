package agent

import (
	"context"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestAutoRecallAppendsHintOnToolError(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Memory: taijicfg.DefaultMemoryConfig()}
	cfg.Memory.GlobalDir = root + "/global-memory"
	svc := memory.NewService(cfg, nil)

	ctx := context.Background()
	_, err := svc.Remember(ctx, memory.RememberRequest{
		Kind:        string(memory.KindError),
		Title:       "Shell permission denied",
		RecallText:  "permission denied when writing file",
		VerifiedFix: "Check directory permissions or use a writable path",
		Fingerprint: "permission:denied:*",
		Scope:       "local",
	})
	if err != nil {
		t.Fatal(err)
	}

	reg := tools.NewRegistry()
	builtin.EnableMemoryOrchestration(reg, svc)

	out := "error: permission denied"
	if svc != nil && shouldAutoRecallMemory("shell", out, nil) {
		if hint := svc.FormatErrorHint(ctx, "shell", out); hint != "" {
			out += "\n\n" + hint
		}
	}

	if !strings.Contains(out, "[memory]") {
		t.Fatalf("expected memory hint in tool output, got: %q", out)
	}
	if !strings.Contains(out, "Shell permission denied") {
		t.Fatalf("expected matched entry title in hint, got: %q", out)
	}
}
