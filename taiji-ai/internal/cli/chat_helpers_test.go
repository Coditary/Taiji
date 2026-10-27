package cli

import (
	"bytes"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/runtime"
)

func TestPrintChatHelpListsCommands(t *testing.T) {
	var buf bytes.Buffer
	printChatHelp(&buf)
	out := buf.String()
	for _, cmd := range []string{"/exit", "/provider", "/compact"} {
		if !strings.Contains(out, cmd) {
			t.Fatalf("missing %q in help:\n%s", cmd, out)
		}
	}
}

func TestResolveChatProviderUsesExplicitID(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy": {Type: "dummy", Model: "alpha"},
		},
	}
	rt, err := runtime.New(t.Context(), cfg, runtime.Options{Quiet: true, Local: true, LocalOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	st := &chatState{rt: rt, providerID: "dummy"}
	id, err := resolveChatProvider(st)
	if err != nil || id != "dummy" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestResolveChatProviderFillsDefault(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy": {Type: "dummy", Model: "alpha"},
		},
	}
	rt, err := runtime.New(t.Context(), cfg, runtime.Options{Quiet: true, Local: true, LocalOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	st := &chatState{rt: rt}
	id, err := resolveChatProvider(st)
	if err != nil {
		t.Fatal(err)
	}
	if id != "dummy" || st.providerID != "dummy" {
		t.Fatalf("id=%q state=%q", id, st.providerID)
	}
}
