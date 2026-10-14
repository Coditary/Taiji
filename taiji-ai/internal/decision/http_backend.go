package decision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const decisionHTTPTimeout = 2 * time.Minute

type httpBackend struct {
	baseURL  string
	driverID string
	model    string
	client   *http.Client
}

func newHTTPBackend(baseURL, driverID, model string) *httpBackend {
	return &httpBackend{
		baseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		driverID: strings.TrimSpace(driverID),
		model:    strings.TrimSpace(model),
		client:   &http.Client{Timeout: decisionHTTPTimeout},
	}
}

func (b *httpBackend) Decide(ctx context.Context, req Request) (*Response, error) {
	if b.baseURL == "" {
		return nil, fmt.Errorf("decision http_url is not configured")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = b.model
	}
	payload := map[string]any{
		"state":     req.Text,
		"questions": toHTTPQuestions(req.Questions),
	}
	if model != "" {
		payload["model"] = model
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/v1/systemone", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("decision http %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return ParseResponse(body, b.driverID, "http")
}

func toHTTPQuestions(questions []Question) map[string]any {
	out := make(map[string]any, len(questions))
	for _, q := range questions {
		spec := map[string]any{
			"type":         string(q.Type),
			"instructions": q.Instructions,
		}
		if len(q.Criteria) > 0 {
			criteria := make(map[string]string, len(q.Criteria))
			for _, c := range q.Criteria {
				criteria[c] = c
			}
			spec["criteria"] = criteria
		}
		out[q.ID] = spec
	}
	return out
}
