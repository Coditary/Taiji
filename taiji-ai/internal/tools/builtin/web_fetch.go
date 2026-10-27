package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	webFetchUserAgent   = "Taiji-Agent/1.0 (+https://github.com/coditary/taiji)"
	webFetchHTTPTimeout = 15 * time.Second
	webFetchMaxCharsDef = 12000
)

type WebFetch struct{}

func (WebFetch) Name() string { return "web_fetch" }

func (WebFetch) Description() string {
	return "Fetch http(s) URL content as markdown (HTML) or plain text/JSON, with SSRF protection for private networks."
}

func (WebFetch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "http or https URL",
			},
			"max_chars": map[string]any{
				"type":        "integer",
				"description": "Max characters to return (default 12000)",
			},
		},
		"required": []string{"url"},
	}
}

func (WebFetch) Run(ctx context.Context, args map[string]any) (string, error) {
	rawURL, _ := args["url"].(string)
	maxChars := intArg(args, "max_chars")
	if maxChars <= 0 {
		maxChars = webFetchMaxCharsDef
	}
	u, err := validateFetchURL(rawURL, toolsSettings.AllowLocalFetchOrDefault())
	if err != nil {
		return "", err
	}
	client := newWebFetchClient(toolsSettings.AllowLocalFetchOrDefault())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", webFetchUserAgent)
	req.Header.Set("Accept", "text/html,application/json,text/plain,*/*")
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
		snip := truncateChars(string(body), 500)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, snip)
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	text := string(body)
	if strings.Contains(ct, "html") || looksLikeHTML(text) {
		if plain, err := htmlToTextViaTaijiTools(ctx, text); err == nil && strings.TrimSpace(plain) != "" {
			text = plain
		} else {
			md, err := htmlToMarkdown(text)
			if err != nil {
				text = stripHTMLTagsRough(text)
			} else {
				text = md
			}
		}
	}
	text = strings.TrimSpace(text)
	if len(text) > maxChars {
		text = text[:maxChars] + "\n[... truncated ...]"
	}
	if text == "" {
		return "(empty response)", nil
	}
	return text, nil
}

func newWebFetchClient(allowLocal bool) *http.Client {
	return &http.Client{
		Timeout: webFetchHTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			_, err := validateFetchURL(req.URL.String(), allowLocal)
			return err
		},
	}
}

func looksLikeHTML(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "<html") || strings.Contains(lower, "<body") || strings.Contains(lower, "<!doctype")
}

func stripHTMLTagsRough(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func truncateChars(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
