package decision

import (
	"context"
	"testing"
)

type stubBackend struct {
	resp *Response
	err  error
}

func (s stubBackend) Decide(context.Context, Request) (*Response, error) {
	return s.resp, s.err
}

func TestServiceDecideAndDescribe(t *testing.T) {
	var svc *Service
	if _, err := svc.Decide(context.Background(), Request{}); err != ErrDisabled {
		t.Fatal("nil service")
	}
	if svc.DescribeBackend() != "" {
		t.Fatal("nil describe")
	}
	svc = &Service{
		Backend:     stubBackend{resp: &Response{}},
		DriverID:    "d1",
		BackendName: "stub",
	}
	resp, err := svc.Decide(context.Background(), Request{Text: "?"})
	if err != nil || resp.Driver != "d1" || resp.Backend != "stub" {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	if svc.DescribeBackend() != "stub/d1" {
		t.Fatal("describe")
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
}
