package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestWebFetchDefinitions(t *testing.T) {
	w := NewWebFetch()
	defs := w.Definitions()
	if len(defs) == 0 {
		t.Fatal("want at least 1 definition, got 0")
	}
	for _, d := range defs {
		if d.Effects != tools.ToolEffectRead {
			t.Errorf("%s: Effects = %v, want Read", d.Name, d.Effects)
		}
	}
}

func TestWebFetch_RedirectToPrivateIPBlocked(t *testing.T) {
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:1/should-not-fetch", http.StatusFound)
	}))
	defer redirect.Close()

	wf := NewWebFetch()
	args, _ := json.Marshal(map[string]string{"url": redirect.URL})
	_, err := wf.Execute(context.Background(), "web_fetch", args)
	if err == nil {
		t.Fatal("expected SSRF block on redirect, got nil error")
	}
	if !strings.Contains(err.Error(), "SSRF") && !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected SSRF-related error, got %v", err)
	}
}
