package events

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Type identifies an agent event.
type Type string

const (
	TypeStep         Type = "step"
	TypeThought      Type = "thought"
	TypeThoughtStart Type = "thought_start"
	TypeThinkDelta   Type = "think_delta"
	TypeContinue     Type = "continue"
	TypeMessage      Type = "message"
	TypeToolCall     Type = "tool_call"
	TypeObservation  Type = "observation"
	TypeError        Type = "error"
	TypeDone         Type = "done"
)

// Event is a single agent loop event.
type Event struct {
	Type       Type           `json:"type"`
	Timestamp  time.Time      `json:"timestamp"`
	SessionID  string         `json:"session_id,omitempty"`
	Step       int            `json:"step,omitempty"`
	Text       string         `json:"text,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`
	ToolID     string         `json:"tool_id,omitempty"`
	Args       string         `json:"args,omitempty"`
	DurationMs int64          `json:"duration_ms,omitempty"`
	IsMCP      bool           `json:"is_mcp,omitempty"`
	MCPServer  string         `json:"mcp_server,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
}

// Writer appends events to a JSONL file and fans out to subscribers.
type Writer struct {
	mu          sync.RWMutex
	path        string
	subscribers []func(Event)
}

// NewWriter creates an event writer.
func NewWriter(path string) *Writer {
	return &Writer{path: path}
}

// Subscribe adds an in-memory subscriber.
func (w *Writer) Subscribe(fn func(Event)) {
	w.mu.Lock()
	w.subscribers = append(w.subscribers, fn)
	w.mu.Unlock()
}

// Emit records an event.
func (w *Writer) Emit(ev Event) error {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	w.mu.RLock()
	subs := append([]func(Event){}, w.subscribers...)
	w.mu.RUnlock()
	for _, fn := range subs {
		fn(ev)
	}
	if w.path == "" {
		return nil
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s\n", raw)
	return err
}
