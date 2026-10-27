package decision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPBackendDecideSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["state"] != "hello" {
			t.Fatalf("state=%v", body["state"])
		}
		_, _ = w.Write([]byte(`{
			"model": "m1",
			"answers": {
				"route": {"type": "choice", "choice": "billing", "confidence": 0.9}
			}
		}`))
	}))
	defer srv.Close()

	b := newHTTPBackend(srv.URL, "laya", "default-model")
	resp, err := b.Decide(context.Background(), Request{
		Text: "hello",
		Questions: []Question{{
			ID:           "route",
			Type:         QuestionChoice,
			Instructions: "Pick",
			Criteria:     []string{"billing", "technical"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Answers["route"].Choice.Choice != "billing" || resp.Backend != "http" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestHTTPBackendErrors(t *testing.T) {
	b := newHTTPBackend("", "d", "")
	if _, err := b.Decide(context.Background(), Request{}); err == nil || !strings.Contains(err.Error(), "http_url") {
		t.Fatalf("empty url: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()
	b = newHTTPBackend(srv.URL, "d", "")
	if _, err := b.Decide(context.Background(), Request{Text: "x"}); err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("bad status: %v", err)
	}
}

func TestToHTTPQuestionsShape(t *testing.T) {
	out := toHTTPQuestions([]Question{{
		ID: "q1", Type: QuestionNoul, Instructions: "Is it urgent?",
	}})
	spec, ok := out["q1"].(map[string]any)
	if !ok || spec["type"] != "noul" {
		t.Fatalf("spec=%v", out)
	}
}
