package session

import "testing"

func TestMCPToolAllowed(t *testing.T) {
	rt := AgentRuntime{
		MCPServers:   []string{"alpha"},
		MCPToolsDeny: []string{"alpha:blocked_tool"},
	}
	if !rt.MCPToolAllowed("alpha", "ok_tool") {
		t.Fatal("expected ok_tool allowed")
	}
	if rt.MCPToolAllowed("alpha", "blocked_tool") {
		t.Fatal("expected blocked_tool denied")
	}
	if rt.MCPToolAllowed("beta", "ok_tool") {
		t.Fatal("expected disabled server")
	}
}
