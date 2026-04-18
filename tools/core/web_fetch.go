package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/swiftdiaries/openharness/tools"
)

// WebFetch fetches a URL and extracts text content.
type WebFetch struct {
	client *http.Client
}

// NewWebFetch creates a new WebFetch tool.
func NewWebFetch() *WebFetch {
	return &WebFetch{client: &http.Client{Timeout: 30 * time.Second}}
}

func (w *WebFetch) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "web_fetch",
			Description: "Fetch a URL and extract its text content (HTML tags stripped).",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "URL to fetch"}
				},
				"required": ["url"]
			}`),
			Effects: tools.ToolEffectRead,
		},
	}
}

func (w *WebFetch) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if name != "web_fetch" {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	if err := tools.CheckSSRF(params.URL); err != nil {
		slog.Warn("security.ssrf_blocked", "url", params.URL, "reason", err.Error())
		return nil, fmt.Errorf("SSRF blocked: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GhostFin/1.0)")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	title := ""
	var content strings.Builder

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			title = extractText(n)
		}
		// Skip script and style elements
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				content.WriteString(text)
				content.WriteString(" ")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	text := content.String()
	const maxLen = 10000
	if len(text) > maxLen {
		text = text[:maxLen]
	}

	result := map[string]string{
		"url":     params.URL,
		"title":   title,
		"content": tools.WrapExternalContent(text),
	}
	return json.Marshal(result)
}
