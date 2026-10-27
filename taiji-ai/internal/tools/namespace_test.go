package tools

import "testing"

func TestNamespaced(t *testing.T) {
	if got := Namespaced("exa", "web_search"); got != "exa__web_search" {
		t.Fatalf("got %q", got)
	}
	if got := Namespaced("", "grep"); got != "grep" {
		t.Fatalf("empty namespace: got %q", got)
	}
	if got := Namespaced("bad__ns", "tool"); got != "bad_ns__tool" {
		t.Fatalf("sanitize: got %q", got)
	}
}

func TestSplitNamespaced(t *testing.T) {
	ns, local, ok := SplitNamespaced("exa__web_search")
	if !ok || ns != "exa" || local != "web_search" {
		t.Fatalf("split: ns=%q local=%q ok=%v", ns, local, ok)
	}
	_, _, ok = SplitNamespaced("grep")
	if ok {
		t.Fatal("expected builtin name not namespaced")
	}
}
