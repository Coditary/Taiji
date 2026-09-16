package rag

import (
	"context"
	"fmt"
	"strings"
)

// Service applies scope rules on top of a pluggable RAG backend.
type Service struct {
	Backend            Backend
	Scope              Scope
	DriverID           string
	DefaultCollection  string
}

// Index writes documents using the scope write root only.
func (s *Service) Index(ctx context.Context, req IndexRequest) (*IndexResult, error) {
	if s == nil || s.Backend == nil {
		return nil, fmt.Errorf("rag service is not configured")
	}
	collection := strings.TrimSpace(req.Collection)
	if collection == "" {
		collection = s.DefaultCollection
	}
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	req.Collection = collection
	root := s.Scope.WriteRoot()
	if req.StoreDir == "" {
		req.StoreDir = s.Scope.WriteStoreDir()
	}
	res, err := s.Backend.Index(ctx, root, req)
	if err != nil {
		return nil, err
	}
	if res != nil {
		res.Scope = scopeLabelForRoot(s.Scope, root)
	}
	return res, nil
}

// Query reads from one or more roots depending on scope mode.
func (s *Service) Query(ctx context.Context, req QueryRequest) ([]Hit, error) {
	if s == nil || s.Backend == nil {
		return nil, fmt.Errorf("rag service is not configured")
	}
	collection := strings.TrimSpace(req.Collection)
	if collection == "" {
		collection = s.DefaultCollection
	}
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	if strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	req.Collection = collection
	if req.TopK <= 0 {
		req.TopK = 5
	}

	roots := s.Scope.ReadRoots()
	var combined []Hit
	for i, sr := range roots {
		if i > 0 && s.Scope.SamePhysicalStore() && sr.Root == roots[0].Root {
			continue
		}
		hits, err := s.Backend.Query(ctx, sr.Root, QueryRequest{
			Collection: req.Collection,
			Query:      req.Query,
			TopK:       req.TopK,
			MinScore:   req.MinScore,
			StoreDir:   sr.StoreDir,
		})
		if err != nil {
			return nil, fmt.Errorf("%s store: %w", sr.Label, err)
		}
		for _, h := range hits {
			h.Scope = sr.Label
			combined = append(combined, h)
		}
	}
	return mergeHits(combined, req.TopK), nil
}

func scopeLabelForRoot(scope Scope, root string) string {
	if root == scope.LocalRoot {
		return "local"
	}
	return "global"
}
