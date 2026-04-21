package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/swiftdiaries/openharness/tools"
)

// disableSSRFForTests opts out of SSRF checks. Set ONLY from *_test.go in
// this package; not exported. Production code path is unaffected.
var disableSSRFForTests bool

// WebFetch fetches a URL and extracts text content.
type WebFetch struct {
	client *http.Client
}

// NewWebFetch creates a new WebFetch tool. The HTTP client re-runs CheckSSRF
// on every redirect hop to prevent SSRF bypass via attacker-controlled 3xx,
// and pins dials to IPs resolved at validate-time via ResolveAndCheck to
// prevent DNS TOCTOU (LookupHost→Dial) rebinding attacks.
func NewWebFetch() *WebFetch {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			var ips []string
			if disableSSRFForTests {
				ips = []string{host}
			} else {
				ips, err = tools.ResolveAndCheck(host)
				if err != nil {
					return nil, fmt.Errorf("SSRF blocked at dial: %w", err)
				}
			}
			var lastErr error
			for _, ip := range ips {
				conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
				if derr == nil {
					return conn, nil
				}
				lastErr = derr
			}
			return nil, lastErr
		},
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			if !disableSSRFForTests {
				if err := tools.CheckSSRF(req.URL.String()); err != nil {
					return fmt.Errorf("SSRF blocked on redirect: %w", err)
				}
			}
			return nil
		},
	}
	return &WebFetch{client: client}
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

	if !disableSSRFForTests {
		if err := tools.CheckSSRF(params.URL); err != nil {
			slog.Warn("security.ssrf_blocked", "url", params.URL, "reason", err.Error())
			return nil, fmt.Errorf("SSRF blocked: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; openharness/0.1)")

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
		"title":   SanitizeRead(title),
		"content": SanitizeExternal(text),
	}
	return json.Marshal(result)
}
