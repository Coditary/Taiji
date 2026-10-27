package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const webSearchTimeout = 20 * time.Second

var (
	braveWebSearchURL    = "https://api.search.brave.com/res/v1/web/search"
	duckDuckGoInstantURL = "https://api.duckduckgo.com/"
)

type WebSearch struct{}

func (WebSearch) Name() string { return "web_search" }

func (WebSearch) Description() string {
	return "Search the public web for current information. Uses Brave Search when BRAVE_API_KEY or tools.brave_search_api_key is set; otherwise DuckDuckGo instant answers (limited)."
}

func (WebSearch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Max results (default from config, usually 8)",
			},
		},
		"required": []string{"query"},
	}
}

func (WebSearch) Run(ctx context.Context, args map[string]any) (string, error) {
	query := strings.TrimSpace(stringArg(args, "query"))
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	maxResults := intArg(args, "max_results")
	if maxResults <= 0 {
		maxResults = toolsSettings.WebSearchMaxResultsOrDefault()
	}
	if key := toolsSettings.BraveSearchKey(); key != "" {
		out, err := braveSearch(ctx, query, maxResults, key)
		if err == nil && strings.TrimSpace(out) != "" {
			return out, nil
		}
		if err != nil {
			// fall through to DDG
			_ = err
		}
	}
	return duckDuckGoSearch(ctx, query, maxResults)
}

func braveSearch(ctx context.Context, query string, limit int, apiKey string) (string, error) {
	u := braveWebSearchURL + "?q=" + url.QueryEscape(query) + fmt.Sprintf("&count=%d", limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)
	client := &http.Client{Timeout: webSearchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("brave search HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	var lines []string
	for i, r := range parsed.Web.Results {
		if i >= limit {
			break
		}
		line := fmt.Sprintf("%d. %s\n   %s\n   %s", i+1, strings.TrimSpace(r.Title), strings.TrimSpace(r.URL), strings.TrimSpace(r.Description))
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("no brave results")
	}
	return strings.Join(lines, "\n\n"), nil
}

func duckDuckGoSearch(ctx context.Context, query string, limit int) (string, error) {
	u := duckDuckGoInstantURL + "?q=" + url.QueryEscape(query) + "&format=json&no_redirect=1&no_html=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: webSearchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return "", err
	}
	var parsed struct {
		Abstract       string `json:"Abstract"`
		AbstractURL    string `json:"AbstractURL"`
		AbstractSource string `json:"AbstractSource"`
		RelatedTopics  []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	var lines []string
	if abs := strings.TrimSpace(parsed.Abstract); abs != "" {
		lines = append(lines, fmt.Sprintf("Summary (%s): %s\n%s", parsed.AbstractSource, abs, parsed.AbstractURL))
	}
	for _, t := range parsed.RelatedTopics {
		if len(lines) >= limit+1 {
			break
		}
		text := strings.TrimSpace(t.Text)
		if text == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s\n  %s", text, strings.TrimSpace(t.FirstURL)))
	}
	if len(lines) == 0 {
		return "No instant answers. Try web_fetch on a specific URL or configure BRAVE_API_KEY for full web search.", nil
	}
	return strings.Join(lines, "\n\n"), nil
}
