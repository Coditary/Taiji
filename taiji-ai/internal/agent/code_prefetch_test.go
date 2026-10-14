package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestExtractCodeSearchTermsRAG(t *testing.T) {
	terms := extractCodeSearchTerms("wo befinden sich solche dinge wie das RAG")
	if len(terms) != 1 || terms[0] != "RAG" {
		t.Fatalf("terms=%v want [RAG]", terms)
	}
}

func TestExtractCodeSearchTermsSpanish(t *testing.T) {
	terms := extractCodeSearchTerms("¿Dónde está el código de MCP en el repo?")
	if len(terms) != 1 || terms[0] != "MCP" {
		t.Fatalf("terms=%v want [MCP]", terms)
	}
}

func TestRunCodePrefetchEmitsGrep(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "internal", "rag", "service.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package rag\n// RAG service\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tools.NewRegistry()
	reg.Register(builtin.Grep{})

	var kinds []string
	preface := runCodePrefetch(context.Background(), RunRequest{
		Tools:   reg,
		WorkDir: root,
	}, "wo ist RAG", func(ev events.Event) {
		kinds = append(kinds, ev.ToolName)
	})
	if preface == "" {
		t.Fatal("expected preface with grep hits")
	}
	if len(kinds) < 2 || kinds[0] != "grep" {
		t.Fatalf("expected grep events, got %v", kinds)
	}
}
