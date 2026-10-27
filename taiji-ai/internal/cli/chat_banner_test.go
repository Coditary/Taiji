package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/session"
)

func TestPrintChatBannerNewAndResumed(t *testing.T) {
	root := t.TempDir()
	sess, err := session.New(root, "My Chat", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	printChatBanner(&buf, sess, "dummy", false)
	if !strings.Contains(buf.String(), "My Chat") || !strings.Contains(buf.String(), "provider: dummy") {
		t.Fatalf("new banner: %s", buf.String())
	}

	buf.Reset()
	printChatBanner(&buf, sess, "dummy", true)
	if !strings.Contains(buf.String(), "resumed session") {
		t.Fatalf("resumed banner: %s", buf.String())
	}
}
