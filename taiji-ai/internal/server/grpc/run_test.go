package grpcserver

import (
	"context"
	"path/filepath"
	"testing"

	taijiv1 "github.com/coditary/taiji/api/proto/v1"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/runtime"
)

type runStreamStub struct {
	taijiv1.TaijiAgent_RunServer
	ctx   context.Context
	events []*taijiv1.AgentEvent
}

func (s *runStreamStub) Context() context.Context { return s.ctx }
func (s *runStreamStub) Send(ev *taijiv1.AgentEvent) error {
	s.events = append(s.events, ev)
	return nil
}

func TestRunDummyAgentEmitsEvents(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy": {Type: "dummy", Model: "alpha"},
		},
		Wuji: taijicfg.WujiConfig{AutoSync: false},
		Skills: taijicfg.SkillsConfig{
			Dirs: []string{filepath.Join(root, ".taiji", "skills")},
		},
		MCP: taijicfg.MCPConfig{ConfigPath: filepath.Join(root, ".wuji", "mcp.json")},
	}
	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Quiet: true, Local: true, LocalOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	s := New(rt)
	stream := &runStreamStub{ctx: context.Background()}
	err = s.Run(&taijiv1.RunRequest{Goal: "Ping", Provider: "dummy"}, stream)
	if err != nil {
		t.Fatal(err)
	}
	if len(stream.events) == 0 {
		t.Fatal("expected agent events on stream")
	}
}
