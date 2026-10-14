package rag

import (
	"context"
	"encoding/json"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

// WujiBackend implements Backend via the Wuji core RunRAG RPC.
type WujiBackend struct {
	endpoint string
	wujiRoot string
	driverID string
	client   *provider.WujiCoreClient
}

func newWujiBackend(cfg *taijicfg.Config) *WujiBackend {
	ragCfg := cfg.ResolvedRAG()
	driverID := strings.TrimSpace(ragCfg.Driver)
	if driverID == "" {
		driverID = taijicfg.DefaultRAGConfig().Driver
	}
	return &WujiBackend{
		endpoint: provider.ResolveTaijiWujiEndpoint(cfg.Root, cfg.Wuji.Endpoint, cfg.Wuji.WujiRoot),
		wujiRoot: provider.ResolveWujiRoot(cfg),
		driverID: driverID,
	}
}

func (b *WujiBackend) Index(ctx context.Context, storeRoot string, req IndexRequest) (*IndexResult, error) {
	client, err := b.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"task":         "index",
		"store_root":   storeRoot,
		"store_dir":    req.StoreDir,
		"collection":   req.Collection,
		"source_paths": req.SourcePaths,
		"recursive":    req.Recursive,
		"glob":         req.Glob,
	}
	if len(req.Exclude) > 0 {
		payload["exclude"] = req.Exclude
	}
	if req.ChunkSize > 0 {
		payload["chunk_size"] = req.ChunkSize
	}
	if req.EmbedModel != "" {
		payload["embed_model"] = req.EmbedModel
	}
	if req.IndexMode != "" {
		payload["index_mode"] = req.IndexMode
	}
	shape, err := client.RunRAG(ctx, b.driverID, payload)
	if err != nil {
		return nil, err
	}
	out := &IndexResult{Collection: req.Collection, Message: "indexed"}
	parseIndexRecords(shape, out)
	return out, nil
}

func (b *WujiBackend) Query(ctx context.Context, storeRoot string, req QueryRequest) ([]Hit, error) {
	client, err := b.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"task":           "query",
		"store_root":     storeRoot,
		"store_dir":      req.StoreDir,
		"collection":     req.Collection,
		"query":          req.Query,
		"top_k":          req.TopK,
		"include_scores": true,
	}
	if req.MinScore > 0 {
		payload["min_score"] = req.MinScore
	}
	shape, err := client.RunRAG(ctx, b.driverID, payload)
	if err != nil {
		return nil, err
	}
	return parseQueryHits(shape)
}

func (b *WujiBackend) clientFor(ctx context.Context) (*provider.WujiCoreClient, error) {
	if b.client != nil {
		return b.client, nil
	}
	client, err := provider.DialWujiCore(ctx, b.endpoint, b.wujiRoot)
	if err != nil {
		return nil, err
	}
	b.client = client
	return b.client, nil
}

// Close releases the gRPC connection.
func (b *WujiBackend) Close() error {
	if b.client == nil {
		return nil
	}
	err := b.client.Close()
	b.client = nil
	return err
}

func parseIndexRecords(shape json.RawMessage, out *IndexResult) {
	var doc struct {
		Records []struct {
			Fields map[string]json.RawMessage `json:"fields"`
		} `json:"records"`
	}
	if err := json.Unmarshal(shape, &doc); err != nil {
		return
	}
	for _, rec := range doc.Records {
		if msg, ok := fieldString(rec.Fields, "message"); ok {
			out.Message = msg
		}
		if n, ok := fieldInt(rec.Fields, "chunks_added"); ok {
			out.ChunksAdded = n
		}
		if n, ok := fieldInt(rec.Fields, "total_chunks"); ok {
			out.TotalChunks = n
		}
	}
}

func parseQueryHits(shape json.RawMessage) ([]Hit, error) {
	var doc struct {
		Records []struct {
			ID     string                       `json:"id"`
			Role   string                       `json:"role"`
			Fields map[string]json.RawMessage `json:"fields"`
		} `json:"records"`
	}
	if err := json.Unmarshal(shape, &doc); err != nil {
		return nil, err
	}
	hits := make([]Hit, 0)
	for _, rec := range doc.Records {
		if rec.Role != "" && rec.Role != "chunk" {
			continue
		}
		hit := Hit{ID: rec.ID}
		if text, ok := fieldString(rec.Fields, "text"); ok {
			hit.Text = text
		}
		if src, ok := fieldString(rec.Fields, "source"); ok {
			hit.Source = src
		}
		if score, ok := fieldFloat(rec.Fields, "score"); ok {
			hit.Score = float32(score)
		}
		if idx, ok := fieldInt(rec.Fields, "chunk_index"); ok {
			hit.ChunkIdx = idx
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

func fieldString(fields map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := fields[key]
	if !ok {
		return "", false
	}
	var v struct {
		S string `json:"s"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", false
	}
	return v.S, v.S != ""
}

func fieldInt(fields map[string]json.RawMessage, key string) (int, bool) {
	raw, ok := fields[key]
	if !ok {
		return 0, false
	}
	var v struct {
		I int64 `json:"i"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, false
	}
	return int(v.I), true
}

func fieldFloat(fields map[string]json.RawMessage, key string) (float64, bool) {
	raw, ok := fields[key]
	if !ok {
		return 0, false
	}
	var v struct {
		F float64 `json:"f"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, false
	}
	return v.F, true
}

var _ Backend = (*WujiBackend)(nil)
