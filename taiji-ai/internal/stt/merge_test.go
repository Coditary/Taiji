package stt

import "testing"

func TestAppendTranscript(t *testing.T) {
	if got := AppendTranscript("hello world", "hello world again"); got != "hello world again" {
		t.Fatalf("extend prefix: %q", got)
	}
	if got := AppendTranscript("foo bar", "baz"); got != "foo bar baz" {
		t.Fatalf("append: %q", got)
	}
}
