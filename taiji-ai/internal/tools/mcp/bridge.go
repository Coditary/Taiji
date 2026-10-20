package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/coditary/taiji/internal/tools"
)

// ServerConfig describes how to launch an MCP server.
type ServerConfig struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	Cwd     string
}

// Bridge connects to MCP servers over stdio and exposes their tools.
type Bridge struct {
	mu      sync.Mutex
	servers map[string]*session
}

type session struct {
	cfg    ServerConfig
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	nextID atomic.Int64
}

// NewBridge creates an MCP tool bridge.
func NewBridge() *Bridge {
	return &Bridge{servers: map[string]*session{}}
}

// Connect starts an MCP server subprocess.
func (b *Bridge) Connect(ctx context.Context, cfg ServerConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.servers[cfg.Name]; ok {
		return nil
	}
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	if cfg.Cwd != "" {
		cmd.Dir = cfg.Cwd
	}
	if len(cfg.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range cfg.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return err
	}
	s := &session{cfg: cfg, cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdoutPipe)}
	if err := s.initialize(ctx); err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	b.servers[cfg.Name] = s
	return nil
}

// ConnectedServers returns names of connected MCP servers.
func (b *Bridge) ConnectedServers() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.servers))
	for name := range b.servers {
		out = append(out, name)
	}
	return out
}

// RegisterTools loads tools from connected MCP servers into the registry.
func (b *Bridge) RegisterTools(r *tools.Registry, serverName string) error {
	b.mu.Lock()
	s, ok := b.servers[serverName]
	b.mu.Unlock()
	if !ok {
		return fmt.Errorf("mcp server %q not connected", serverName)
	}
	list, err := s.listTools(context.Background())
	if err != nil {
		return err
	}
	for _, t := range list {
		r.Register(&mcpTool{bridge: b, server: serverName, def: t})
	}
	return nil
}

// Close shuts down all MCP sessions.
func (b *Bridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var first error
	for name, s := range b.servers {
		if err := s.close(); err != nil && first == nil {
			first = fmt.Errorf("%s: %w", name, err)
		}
		delete(b.servers, name)
	}
	return first
}

type mcpToolDef struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// ServerTool identifies a tool backed by an MCP server (or operating on one).
type ServerTool interface {
	tools.Tool
	MCPServerName() string
}

type mcpTool struct {
	bridge *Bridge
	server string
	def    mcpToolDef
}

func (t *mcpTool) Name() string          { return t.def.Name }
func (t *mcpTool) MCPServerName() string { return t.server }
func (t *mcpTool) Description() string   { return t.def.Description }
func (t *mcpTool) Parameters() map[string]any {
	if t.def.InputSchema != nil {
		return t.def.InputSchema
	}
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func (t *mcpTool) Run(ctx context.Context, args map[string]any) (string, error) {
	t.bridge.mu.Lock()
	s, ok := t.bridge.servers[t.server]
	t.bridge.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("mcp server %q not connected", t.server)
	}
	return s.callTool(ctx, t.def.Name, args)
}

func (s *session) initialize(ctx context.Context) error {
	_, err := s.request(ctx, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"resources": map[string]any{},
		},
		"clientInfo": map[string]any{
			"name":    "taiji",
			"version": "0.1.0",
		},
	})
	if err != nil {
		return err
	}
	return s.notify("notifications/initialized", map[string]any{})
}

func (s *session) listTools(ctx context.Context) ([]mcpToolDef, error) {
	resp, err := s.request(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	toolsRaw, _ := resp["tools"].([]any)
	out := make([]mcpToolDef, 0, len(toolsRaw))
	for _, item := range toolsRaw {
		m, _ := item.(map[string]any)
		name, _ := m["name"].(string)
		desc, _ := m["description"].(string)
		schema, _ := m["inputSchema"].(map[string]any)
		out = append(out, mcpToolDef{Name: name, Description: desc, InputSchema: schema})
	}
	return out, nil
}

func (s *session) listResources(ctx context.Context) ([]mcpResourceDef, error) {
	resp, err := s.request(ctx, "resources/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	resourcesRaw, _ := resp["resources"].([]any)
	out := make([]mcpResourceDef, 0, len(resourcesRaw))
	for _, item := range resourcesRaw {
		m, _ := item.(map[string]any)
		uri, _ := m["uri"].(string)
		name, _ := m["name"].(string)
		desc, _ := m["description"].(string)
		mime, _ := m["mimeType"].(string)
		out = append(out, mcpResourceDef{
			URI:         uri,
			Name:        name,
			Description: desc,
			MimeType:    mime,
		})
	}
	return out, nil
}

func (s *session) readResource(ctx context.Context, uri string) (string, error) {
	resp, err := s.request(ctx, "resources/read", map[string]any{"uri": uri})
	if err != nil {
		return "", err
	}
	return formatMCPContents(resp["contents"]), nil
}

func formatMCPContents(v any) string {
	items, _ := v.([]any)
	var parts []string
	for _, item := range items {
		m, _ := item.(map[string]any)
		if text, ok := m["text"].(string); ok {
			parts = append(parts, text)
			continue
		}
		if blob, ok := m["blob"].(string); ok && blob != "" {
			parts = append(parts, blob)
		}
	}
	return strings.Join(parts, "\n")
}

func (s *session) callTool(ctx context.Context, name string, args map[string]any) (string, error) {
	resp, err := s.request(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}
	if isError, _ := resp["isError"].(bool); isError {
		return formatMCPContent(resp["content"]), fmt.Errorf("mcp tool error")
	}
	return formatMCPContent(resp["content"]), nil
}

func formatMCPContent(v any) string {
	items, _ := v.([]any)
	var parts []string
	for _, item := range items {
		m, _ := item.(map[string]any)
		if text, ok := m["text"].(string); ok {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func (s *session) request(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	id := s.nextID.Add(1)
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
	if err := s.write(req); err != nil {
		return nil, err
	}
	for {
		msg, err := s.read()
		if err != nil {
			return nil, err
		}
		if msg["method"] != nil {
			continue // ignore notifications for MVP
		}
		if messageIDMatches(msg["id"], id) {
			if errObj, ok := msg["error"].(map[string]any); ok {
				return nil, fmt.Errorf("mcp %s: %v", method, errObj["message"])
			}
			result, _ := msg["result"].(map[string]any)
			return result, nil
		}
	}
}

func (s *session) notify(method string, params map[string]any) error {
	return s.write(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func messageIDMatches(raw any, want int64) bool {
	switch v := raw.(type) {
	case float64:
		return int64(v) == want
	case int:
		return int64(v) == want
	case int64:
		return v == want
	case json.Number:
		n, err := v.Int64()
		return err == nil && n == want
	default:
		return false
	}
}

func (s *session) write(v map[string]any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.stdin, "%s\n", raw)
	return err
}

func (s *session) read() (map[string]any, error) {
	for {
		line, err := s.stdout.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			return s.readContentLengthFrame(line)
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, err
		}
		return msg, nil
	}
}

func (s *session) readContentLengthFrame(firstHeader string) (map[string]any, error) {
	headers := map[string]string{}
	parts := strings.SplitN(firstHeader, ":", 2)
	if len(parts) == 2 {
		headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}
	for {
		line, err := s.stdout.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
		}
	}
	length := 0
	if v, ok := headers["content-length"]; ok {
		_, _ = fmt.Sscanf(v, "%d", &length)
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(s.stdout, body); err != nil {
		return nil, err
	}
	var msg map[string]any
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *session) close() error {
	_ = s.stdin.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	return s.cmd.Wait()
}
