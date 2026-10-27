package grpcserver

import (
	"context"
	"testing"
	"time"

	taijiv1 "github.com/coditary/taiji/api/proto/v1"
	"github.com/coditary/taiji/internal/events"
)

func TestPing(t *testing.T) {
	s := New(nil)
	resp, err := s.Ping(context.Background(), &taijiv1.PingRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Ok || resp.Version == "" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestStopNilGRPC(t *testing.T) {
	s := New(nil)
	s.Stop()
}

func TestToProtoEvent(t *testing.T) {
	ts := time.UnixMilli(1_700_000_000_000)
	ev := events.Event{
		Type:      events.TypeToolCall,
		Step:      2,
		Text:      "hi",
		ToolName:  "read_file",
		ToolID:    "tc1",
		Args:      "{}",
		SessionID: "sess",
		Timestamp: ts,
	}
	out := toProtoEvent(ev)
	if out.Type != "tool_call" || out.Step != 2 || out.ToolName != "read_file" || out.SessionId != "sess" {
		t.Fatalf("proto=%+v", out)
	}
	if out.TimestampUnixMs != ts.UnixMilli() {
		t.Fatalf("ts=%d", out.TimestampUnixMs)
	}
}
