package mcp

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestReadContentLengthFrame(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`
	var buf strings.Builder
	fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n%s", len(body), body)
	s := &session{stdout: bufio.NewReader(strings.NewReader(buf.String()))}
	msg, err := s.readContentLengthFrame("Content-Length: " + fmt.Sprint(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	if msg["id"].(float64) != 1 {
		t.Fatalf("msg=%v", msg)
	}
}

func TestSessionReadContentLengthThenLine(t *testing.T) {
	lineBody := `{"jsonrpc":"2.0","method":"ping"}`
	framed := `{"jsonrpc":"2.0","id":2,"result":{"tools":[]}}`
	var w strings.Builder
	fmt.Fprintf(&w, "Content-Length: %d\r\n\r\n%s", len(framed), framed)
	w.WriteString(lineBody)
	w.WriteString("\n")
	s := &session{stdout: bufio.NewReader(strings.NewReader(w.String()))}
	first, err := s.read()
	if err != nil || first["id"].(float64) != 2 {
		t.Fatalf("framed: err=%v msg=%v", err, first)
	}
	second, err := s.read()
	if err != nil || second["method"] != "ping" {
		t.Fatalf("line: err=%v msg=%v", err, second)
	}
	if _, err := s.read(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}
