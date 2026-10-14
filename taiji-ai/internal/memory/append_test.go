package memory

import (
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestSystemPromptAppendix(t *testing.T) {
	svc := NewService(&taijicfg.Config{Root: "/tmp"}, nil)
	appendix := SystemPromptAppendix(svc)
	if appendix == "" {
		t.Fatal("expected appendix")
	}
	if !strings.Contains(appendix, "memory_recall") || !strings.Contains(appendix, "memory_remember") || !strings.Contains(appendix, "memory_revise") {
		t.Fatalf("missing tools in appendix: %q", appendix)
	}
	if !strings.Contains(appendix, "Lookup before claiming ignorance") {
		t.Fatalf("missing lookup guidance: %q", appendix)
	}
	if !strings.Contains(appendix, "Memory kinds (catalog)") {
		t.Fatalf("missing kinds catalog: %q", appendix)
	}
	if !strings.Contains(appendix, "persona") {
		t.Fatalf("missing persona kind: %q", appendix)
	}
}
