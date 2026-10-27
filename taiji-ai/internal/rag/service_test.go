package rag_test

import (
	"context"
	"errors"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

type stubBackend struct {
	indexErr error
	queryErr error
	lastQuery rag.QueryRequest
}

func (s *stubBackend) Index(ctx context.Context, storeRoot string, req rag.IndexRequest) (*rag.IndexResult, error) {
	if s.indexErr != nil {
		return nil, s.indexErr
	}
	return &rag.IndexResult{Collection: req.Collection, ChunksAdded: 2}, nil
}

func (s *stubBackend) Query(ctx context.Context, storeRoot string, req rag.QueryRequest) ([]rag.Hit, error) {
	s.lastQuery = req
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	return []rag.Hit{{Text: "hit", Score: 0.9, Scope: "local"}}, nil
}

func TestServiceIndexAndQuery(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{WujiRoot: t.TempDir()}}
	svc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	backend := &stubBackend{}
	svc.Backend = backend

	res, err := svc.Index(context.Background(), rag.IndexRequest{SourcePaths: []string{"a.md"}})
	if err != nil || res == nil || res.Collection != svc.DefaultCollection {
		t.Fatalf("index: res=%+v err=%v", res, err)
	}

	hits, err := svc.Query(context.Background(), rag.QueryRequest{Query: "needle"})
	if err != nil || len(hits) == 0 || hits[0].Text != "hit" {
		t.Fatalf("query: hits=%v err=%v", hits, err)
	}
	if backend.lastQuery.TopK != 5 {
		t.Fatalf("default topk=%d", backend.lastQuery.TopK)
	}
}

func TestServiceValidationErrors(t *testing.T) {
	var nilSvc *rag.Service
	if _, err := nilSvc.Index(context.Background(), rag.IndexRequest{}); err == nil {
		t.Fatal("nil service index")
	}
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{WujiRoot: t.TempDir()}}
	svc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	svc.Backend = &stubBackend{}
	if _, err := svc.Query(context.Background(), rag.QueryRequest{}); err == nil {
		t.Fatal("empty query")
	}
	svc.DefaultCollection = ""
	if _, err := svc.Index(context.Background(), rag.IndexRequest{SourcePaths: []string{"x"}}); err == nil {
		t.Fatal("missing collection")
	}
}

func TestServiceQueryBackendError(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{WujiRoot: t.TempDir()}}
	svc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	svc.Backend = &stubBackend{queryErr: errors.New("backend down")}
	if _, err := svc.Query(context.Background(), rag.QueryRequest{Query: "x"}); err == nil {
		t.Fatal("expected backend error")
	}
}
