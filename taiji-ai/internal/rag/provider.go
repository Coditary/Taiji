package rag

import "context"

// Backend runs RAG operations against a concrete store (Wuji today; others later).
type Backend interface {
	Index(ctx context.Context, storeRoot string, req IndexRequest) (*IndexResult, error)
	Query(ctx context.Context, storeRoot string, req QueryRequest) ([]Hit, error)
}
