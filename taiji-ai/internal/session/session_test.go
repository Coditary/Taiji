package session

import "testing"

func TestSanitizeName(t *testing.T) {
	tests := map[string]string{
		`  "Hello World"  `: "Hello World",
		"'Title'":        "Title",
		"":               "",
	}
	for in, want := range tests {
		if got := sanitizeName(in); got != want {
			t.Fatalf("sanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSetNameMarksManual(t *testing.T) {
	root := t.TempDir()
	sess, err := New(root, "id1", "", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.SetName("Manual title", true); err != nil {
		t.Fatal(err)
	}
	if !sess.State.NameManual {
		t.Fatal("expected manual name flag")
	}
	if sess.NeedsAutoName() {
		t.Fatal("manual session should not need auto name")
	}
}

func TestNeedsAutoNameWhenUnset(t *testing.T) {
	root := t.TempDir()
	sess, err := New(root, "id2", "", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if !sess.NeedsAutoName() {
		t.Fatal("expected auto name needed")
	}
}

func TestLabelWithAndWithoutName(t *testing.T) {
	root := t.TempDir()
	sess, err := New(root, "abc", "", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if sess.Label() != "abc" {
		t.Fatalf("label = %q", sess.Label())
	}
	if err := sess.SetName("My chat", false); err != nil {
		t.Fatal(err)
	}
	if sess.Label() != "My chat (abc)" {
		t.Fatalf("label = %q", sess.Label())
	}
}
