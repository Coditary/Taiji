package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"google.golang.org/grpc"

	wujiv1 "github.com/coditary/wuji-core/api/proto/v1"
)

type stubWujiCoreClient struct {
	callResp *wujiv1.CallResponse
	callErr  error
}

func (s *stubWujiCoreClient) Call(ctx context.Context, in *wujiv1.CallRequest, opts ...grpc.CallOption) (*wujiv1.CallResponse, error) {
	return s.callResp, s.callErr
}

func (s *stubWujiCoreClient) GenerateTextStream(ctx context.Context, in *wujiv1.CallRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[wujiv1.TextStreamChunk], error) {
	return nil, fmt.Errorf("stream not used in test")
}

func (s *stubWujiCoreClient) Ping(ctx context.Context, in *wujiv1.PingRequest, opts ...grpc.CallOption) (*wujiv1.PingResponse, error) {
	return &wujiv1.PingResponse{}, nil
}

func (s *stubWujiCoreClient) PutFiles(ctx context.Context, in *wujiv1.PutFilesRequest, opts ...grpc.CallOption) (*wujiv1.PutFilesResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *stubWujiCoreClient) GetFiles(ctx context.Context, in *wujiv1.GetFilesRequest, opts ...grpc.CallOption) (*wujiv1.GetFilesResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestWujiProviderComplete(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"text":          "hello wuji",
		"tokens_used":   4,
		"input_tokens":  2,
		"finish_reason": "stop",
		"thinking":      "hmm",
	})
	p := &wujiProvider{
		id:          "wuji-test",
		driverID:    "dummy",
		model:       "test-model",
		maxTokens:   128,
		temperature: 0.2,
		client: &WujiCoreClient{
			client: &stubWujiCoreClient{
				callResp: &wujiv1.CallResponse{PayloadJson: payload},
			},
		},
	}
	resp, err := p.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello wuji" || resp.FinishReason != "stop" {
		t.Fatalf("resp=%+v", resp)
	}
	if len(resp.ThinkingBlocks) != 1 || resp.ThinkingBlocks[0].Text != "hmm" {
		t.Fatalf("thinking=%+v", resp.ThinkingBlocks)
	}
	if p.ID() != "wuji-test" || p.Type() != "wuji" {
		t.Fatalf("id/type mismatch")
	}
}

func TestWujiHelpersFirstNonEmpty(t *testing.T) {
	if got := firstPositive(0, 9); got != 9 {
		t.Fatalf("firstPositive=%d", got)
	}
	if got := firstPositiveFloat(0, 0.5); got != 0.5 {
		t.Fatalf("firstPositiveFloat=%v", got)
	}
	if got := firstNonEmpty("", "x", "y"); got != "x" {
		t.Fatalf("firstNonEmpty=%q", got)
	}
}
