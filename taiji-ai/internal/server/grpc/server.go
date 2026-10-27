package grpcserver

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	taijiv1 "github.com/coditary/taiji/api/proto/v1"
	"github.com/coditary/taiji/internal/agent"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
)

// Server hosts the Taiji gRPC API.
type Server struct {
	taijiv1.UnimplementedTaijiAgentServer
	rt     *runtime.Runtime
	grpc   *grpc.Server
	mu     sync.Mutex
	cancel map[string]context.CancelFunc
}

// New creates a gRPC server.
func New(rt *runtime.Runtime) *Server {
	return &Server{rt: rt, cancel: map[string]context.CancelFunc{}}
}

// Serve listens on addr.
func (s *Server) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.grpc = grpc.NewServer()
	taijiv1.RegisterTaijiAgentServer(s.grpc, s)
	return s.grpc.Serve(lis)
}

// Stop gracefully stops the server.
func (s *Server) Stop() {
	if s.grpc != nil {
		s.grpc.GracefulStop()
	}
}

func (s *Server) Ping(context.Context, *taijiv1.PingRequest) (*taijiv1.PingResponse, error) {
	return &taijiv1.PingResponse{Ok: true, Version: "0.1.0"}, nil
}

func (s *Server) Run(req *taijiv1.RunRequest, stream taijiv1.TaijiAgent_RunServer) error {
	ctx := stream.Context()
	runID := fmt.Sprintf("%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel[runID] = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.cancel, runID)
		s.mu.Unlock()
		cancel()
	}()

	var sess *session.Session
	if req.GetSessionId() != "" {
		sess, _ = session.Load(s.rt.Config.Root, req.GetSessionId())
	}
	if sess == nil {
		var err error
		sess, err = session.New(s.rt.Config.Root, "", req.GetProvider(), req.GetGoal())
		if err != nil {
			return err
		}
	}

	s.rt.Questions.SetPrompter(nil)

	_, err := agent.Run(ctx, agent.RunRequest{
		Goal:       req.GetGoal(),
		ProviderID: req.GetProvider(),
		SkillNames: req.GetSkills(),
		Session:    sess,
		Providers:  s.rt.Providers,
		Tools:      s.rt.Tools,
		Skills:        s.rt.Skills,
		SkillsCatalog: s.rt.SkillsCatalog,
		Config:        s.rt.Config,
		RAG:        s.rt.RAG,
		Memory:     s.rt.Memory,
		MCP:              s.rt.MCP,
		DecisionProfiles: s.rt.DecisionProfiles,
		PackIndex:        s.rt.PackIndex,
		Hooks:            s.rt.Hooks,
		Questions:        s.rt.Questions,
		EventHandler: func(ev events.Event) {
			_ = stream.Send(toProtoEvent(ev))
		},
	})
	if err != nil {
		return err
	}
	return sess.Save()
}

func toProtoEvent(ev events.Event) *taijiv1.AgentEvent {
	return &taijiv1.AgentEvent{
		Type:            string(ev.Type),
		Step:            int32(ev.Step),
		Text:            ev.Text,
		ToolName:        ev.ToolName,
		ToolId:          ev.ToolID,
		Args:            ev.Args,
		SessionId:       ev.SessionID,
		TimestampUnixMs: ev.Timestamp.UnixMilli(),
	}
}
