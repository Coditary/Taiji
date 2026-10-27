package tools

import (
	"context"
	"strings"
	"testing"
)

type stubTool struct{ name string }

func (s stubTool) Name() string        { return s.name }
func (s stubTool) Description() string { return "stub" }
func (s stubTool) Parameters() map[string]any {
	return map[string]any{"type": "object"}
}
func (s stubTool) Run(ctx context.Context, args map[string]any) (string, error) {
	return "ok", nil
}

func TestFilterRegistrySubset(t *testing.T) {
	main := NewRegistry()
	main.Register(stubTool{name: "a"})
	main.Register(stubTool{name: "b"})
	sub := FilterRegistry(main, []string{"b", "missing", "a"})
	if _, ok := sub.Get("a"); !ok {
		t.Fatal("missing a")
	}
	if _, ok := sub.Get("b"); !ok {
		t.Fatal("missing b")
	}
	if n := len(sub.List()); n != 2 {
		t.Fatalf("list=%d", n)
	}
}

type stubMetaTool struct{ stubTool }

func (s stubMetaTool) RunWithMetadata(_ context.Context, _ map[string]any) (RunResult, error) {
	return RunResult{Text: "meta", Data: map[string]any{"k": 1}}, nil
}

type hiddenTool struct{ stubTool }

func (hiddenTool) HideFromCatalog() bool { return true }

func TestRegistryRunAndDefinitions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubTool{name: "plain"})
	reg.Register(stubMetaTool{stubTool{name: "meta"}})
	reg.Register(hiddenTool{stubTool{name: "hidden"}})

	defs := reg.Definitions()
	if len(defs) != 2 {
		t.Fatalf("defs=%d", len(defs))
	}
	res, err := reg.Run(context.Background(), "plain", `{"x":1}`)
	if err != nil || res.Text != "ok" {
		t.Fatalf("plain: %v %q", err, res.Text)
	}
	res, err = reg.Run(context.Background(), "meta", "")
	if err != nil || res.Text != "meta" || res.Data["k"] != 1 {
		t.Fatalf("meta: %+v err=%v", res, err)
	}
	if _, err := reg.Run(context.Background(), "missing", ""); err == nil {
		t.Fatal("missing tool")
	}
	if _, err := reg.Run(context.Background(), "plain", `{bad`); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("bad json: %v", err)
	}
}

func TestStringsTrim(t *testing.T) {
	if stringsTrim("  hi \n") != "hi" {
		t.Fatal("trim")
	}
}
