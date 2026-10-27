package builtin

import (
	"strings"
	"testing"
)

func TestHTMLToMarkdownHeadingsAndParagraph(t *testing.T) {
	raw := `<html><body><h1>Title</h1><p>Hello <strong>world</strong>.</p></body></html>`
	md, err := htmlToMarkdown(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Title") || !strings.Contains(md, "Hello") || !strings.Contains(md, "world") {
		t.Fatalf("md=%q", md)
	}
}

func TestHTMLToMarkdownSkipsScript(t *testing.T) {
	raw := `<html><body><script>alert(1)</script><p>visible</p></body></html>`
	md, err := htmlToMarkdown(raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(md, "alert") {
		t.Fatalf("script leaked: %q", md)
	}
	if !strings.Contains(md, "visible") {
		t.Fatalf("md=%q", md)
	}
}
