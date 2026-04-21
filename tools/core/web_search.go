package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/swiftdiaries/openharness/tools"
)

// WebSearch searches the web via DuckDuckGo HTML.
type WebSearch struct {
	client *http.Client
}

// NewWebSearch creates a new WebSearch tool.
func NewWebSearch() *WebSearch {
	return &WebSearch{client: &http.Client{Timeout: 15 * time.Second}}
}

func (w *WebSearch) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "web_search",
			Description: "Search the web using DuckDuckGo. Returns titles, URLs, and snippets.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query"},
					"limit": {"type": "integer", "description": "Max results (default 5)"}
				},
				"required": ["query"]
			}`),
			Effects: tools.ToolEffectRead,
		},
	}
}

func (w *WebSearch) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if name != "web_search" {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	var params struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if params.Limit <= 0 {
		params.Limit = 5
	}

	u := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(params.Query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; openharness/0.1)")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	type Result struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	}

	var results []Result
	var crawl func(*html.Node)
	crawl = func(n *html.Node) {
		if len(results) >= params.Limit {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "result__a") {
					var href string
					for _, a := range n.Attr {
						if a.Key == "href" {
							href = a.Val
							break
						}
					}
					title := extractText(n)
					// Find sibling snippet
					snippet := ""
					if n.Parent != nil && n.Parent.Parent != nil {
						snippet = findSnippet(n.Parent.Parent)
					}
					if title != "" && href != "" {
						results = append(results, Result{
							Title:   SanitizeRead(title),
							URL:     href,
							Snippet: SanitizeExternal(snippet),
						})
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			crawl(c)
		}
	}
	crawl(doc)

	return json.Marshal(results)
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return strings.TrimSpace(sb.String())
}

func findSnippet(n *html.Node) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			for _, attr := range c.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "result__snippet") {
					return extractText(c)
				}
			}
		}
	}
	return ""
}
