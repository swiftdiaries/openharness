# Tools Review Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

## Pipeline Tracking

| Field | Value |
|-------|-------|
| Epic | `openharness-bv0` |
| Status | complete |
| Completed | 2026-04-21 |
| Worktree | `/Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/.claude/worktrees/beautiful-hermann-3d3ef2` (branch: `claude/beautiful-hermann-3d3ef2`) |

| Task | Beads ID | Status |
|------|----------|--------|
| Task 1: SSRF redirect bypass in web_fetch | `openharness-bv0.1` | closed |
| Task 2: DNS TOCTOU — pin resolved IP through dialer | `openharness-bv0.2` | closed |
| Task 3: Expand exec metacharacter blocklist | `openharness-bv0.3` | closed |
| Task 4: Scrub credentials on Read-effect tool outputs | `openharness-bv0.5` | closed |
| Task 5: Wrap external content in web_search + knowledge_graph | `openharness-bv0.6` | closed |
| Task 6: Reject empty WorkspacePath in Register | `openharness-bv0.4` | closed |
| Task 7: Split exec into read-only and mutating definitions | `openharness-bv0.7` | closed |
| Task 8: Tighten memory + tasks file permissions to 0600 | `openharness-bv0.8` | closed |
| Task 9: Registry name-collision guard + delete deferredActivator | `openharness-bv0.9` | closed |
| Task 10: Delete unused ToolVisibility | `openharness-bv0.10` | closed |
| Task 11: Extract filesystem write-guard as injectable callback | `openharness-bv0.11` | closed |
| Task 12: Delete DefaultWorkspace (~/.ghostfin/workspace) | `openharness-bv0.12` | closed (folded into T11 commit) |
| Task 13: Rename GhostFin User-Agent to openharness | `openharness-bv0.13` | closed |
| Task 14: Add tools/README.md for tool authors | `openharness-bv0.14` | closed |
| Task 15: Add httptest round-trip tests for web_fetch + web_search | `openharness-bv0.15` | closed |
| Task 16: Fix dropped errors and shadowed err | `openharness-bv0.16` | closed |
| Task 17: Add TODO(plan-4) marker in ask_user test | `openharness-bv0.17` | closed |

**Follow-ups filed during execution (not blocking):**
- (none — all review findings were fixed inline or were already in-scope of a planned task)

**Goal:** Address security and hygiene findings from the 2026-04-21 daily code review of the Layer 2 Plan 3 (tools/core) port.

**Architecture:** Work is grouped in four phases by priority. Phase 1 (Critical Security) must land before any network-facing deployment; Phases 2–4 are hardening, framework hygiene, and polish. Every change is test-first. Each task commits independently so the history stays bisectable.

**Tech Stack:** Go 1.25, `net/http`, `net`, `net/http/httptest`, `testing`.

**Worktree:** `/Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/.claude/worktrees/beautiful-hermann-3d3ef2`

**Reference:** Review findings from 2026-04-21 daily-code-review scheduled task.

---

## File Structure

Files this plan creates or modifies (see individual tasks for line-precise scope):

**Phase 1 — Critical Security**
- `tools/ssrf.go` — pin resolved IPs; expose resolver to Dialer.
- `tools/ssrf_test.go` — new redirect/TOCTOU test cases.
- `tools/core/web_fetch.go` — CheckRedirect hook; SSRF-aware Dialer.
- `tools/core/web_fetch_test.go` — httptest redirect-bypass test.
- `tools/core/exec.go` — expand metacharacter blocklist (newline, glob, brace, tilde).
- `tools/core/exec_test.go` — coverage for each new blocked metachar.

**Phase 2 — Hardening**
- `tools/scrub.go` — already has `ScrubCredentials`; no code change.
- `tools/core/web_fetch.go`, `web_search.go`, `knowledge_graph.go`, `memory.go`, `filesystem.go`, `exec.go` — apply `ScrubCredentials` to Read-effect outputs (or via helper).
- `tools/core/external_output.go` (new) — shared scrub+wrap helper for Read-effect tools.
- `tools/core/web_search.go`, `knowledge_graph.go` — wrap external content with `WrapExternalContent`.
- `tools/core/core.go` — reject empty `WorkspacePath` in `Register`.
- `tools/core/core_test.go` — assert Register fails on empty WorkspacePath.

**Phase 3 — Framework hygiene**
- `tools/core/exec.go` — split `exec` into two definitions: read-only (Read/Neutral) and mutating (Mutate).
- `tools/core/exec_test.go` — assert Effects for each split definition.
- `tools/core/memory.go`, `tools/core/tasks.go` — file permissions 0644 → 0600.
- `tools/registry.go` — name-collision guard + delete `deferredActivator` field/method.
- `tools/tool.go` — delete `ToolVisibility` enum + struct field.
- `tools/tool_test.go` — remove visibility tests.
- `tools/core/filesystem.go` — replace `agentType`/`selfEvolve`/`predefinedImmutableFiles` with injectable `WriteGuardFunc`; delete `DefaultWorkspace()`.
- `tools/core/filesystem_test.go` — rewrite write-guard tests to exercise the callback.
- `tools/core/web_fetch.go`, `web_search.go` — rename `GhostFin/1.0` User-Agent to `openharness/0.x`.

**Phase 4 — Polish**
- `tools/README.md` (new) — tool authoring guide.
- `tools/core/web_fetch_test.go`, `web_search_test.go` — `httptest.Server` round-trip tests.
- `tools/core/tasks.go`, `filesystem.go`, `knowledge_graph.go` — fix dropped error returns + shadowed `err`.
- `tools/core/ask_user_test.go` — add `TODO(plan-4)` marker.

---

# Phase 1 — Critical Security

## Task 1: SSRF redirect bypass in web_fetch

**Why:** `http.Client` follows up to 10 redirects with no `CheckRedirect` hook, so an attacker-controlled public URL can 302 to `http://169.254.169.254/...` or `http://127.0.0.1/...` and bypass `CheckSSRF`.

**Files:**
- Modify: `tools/core/web_fetch.go:23-25, 67` (add `CheckRedirect` in constructor; SSRF-check every hop)
- Modify: `tools/core/web_fetch_test.go` (new test)

- [ ] **Step 1: Write the failing test**

Append to `tools/core/web_fetch_test.go`:

```go
package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebFetch_RedirectToPrivateIPBlocked(t *testing.T) {
	// Server that redirects to a private-IP URL.
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./tools/core/ -run TestWebFetch_RedirectToPrivateIPBlocked -v
```

Expected: FAIL — redirect is followed and the fetch either succeeds or fails with a non-SSRF connection error.

- [ ] **Step 3: Implement the fix**

In `tools/core/web_fetch.go`, replace the `NewWebFetch` constructor (currently lines 22-25):

```go
// NewWebFetch creates a new WebFetch tool. The HTTP client re-runs CheckSSRF
// on every redirect hop to prevent SSRF bypass via attacker-controlled 3xx.
func NewWebFetch() *WebFetch {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			if err := tools.CheckSSRF(req.URL.String()); err != nil {
				return fmt.Errorf("SSRF blocked on redirect: %w", err)
			}
			return nil
		},
	}
	return &WebFetch{client: client}
}
```

- [ ] **Step 4: Run the test**

```bash
go test ./tools/core/ -run TestWebFetch_RedirectToPrivateIPBlocked -v
```

Expected: PASS.

- [ ] **Step 5: Run the full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/core/web_fetch.go tools/core/web_fetch_test.go
git commit -m "tools/core: re-check SSRF on web_fetch redirect hops"
```

---

## Task 2: DNS TOCTOU — pin resolved IP through the dialer

**Why:** `CheckSSRF` calls `net.LookupHost` at validate time; `http.Client` re-resolves at dial time. Short-TTL DNS or split-horizon can serve public→private between the two resolutions. Fix: resolve once, dial the resolved IP, keep the Host header for TLS SNI.

**Files:**
- Modify: `tools/ssrf.go` (add `ResolveAndCheck` returning addrs)
- Modify: `tools/core/web_fetch.go` (custom `Transport.DialContext` using pinned addrs)
- Modify: `tools/ssrf_test.go` (new test for ResolveAndCheck)
- Modify: `tools/core/web_fetch_test.go` (new test asserting dial uses resolved IP)

- [ ] **Step 1: Write the failing test for the resolver**

Append to `tools/ssrf_test.go`:

```go
func TestResolveAndCheck_ReturnsPublicAddrs(t *testing.T) {
	addrs, err := ResolveAndCheck("example.com")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(addrs) == 0 {
		t.Fatal("expected at least one public addr")
	}
}

func TestResolveAndCheck_RejectsPrivateResolve(t *testing.T) {
	// localhost resolves to 127.0.0.1 → private; must reject.
	_, err := ResolveAndCheck("localhost")
	if err == nil {
		t.Fatal("expected rejection for localhost")
	}
}

func TestResolveAndCheck_IPLiteralPublic(t *testing.T) {
	addrs, err := ResolveAndCheck("8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
		t.Fatalf("expected [8.8.8.8], got %v", addrs)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./tools/ -run TestResolveAndCheck -v
```

Expected: FAIL — `ResolveAndCheck` undefined.

- [ ] **Step 3: Implement `ResolveAndCheck`**

In `tools/ssrf.go`, add after `CheckSSRF`:

```go
// ResolveAndCheck validates a hostname-or-IP and returns the resolved public
// IPs. Callers should pin HTTP dials to these addresses to prevent DNS TOCTOU
// (LookupHost→Dial) rebinding attacks.
func ResolveAndCheck(hostname string) ([]string, error) {
	if hostname == "" {
		return nil, fmt.Errorf("missing hostname")
	}
	if isBlockedHostname(hostname) {
		return nil, fmt.Errorf("blocked hostname: %s", hostname)
	}
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(hostname) {
			return nil, fmt.Errorf("private IP address not allowed: %s", hostname)
		}
		return []string{hostname}, nil
	}
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed for %s: %w", hostname, err)
	}
	for _, addr := range addrs {
		if isPrivateIP(addr) {
			return nil, fmt.Errorf("hostname %s resolves to private IP %s", hostname, addr)
		}
	}
	return addrs, nil
}
```

- [ ] **Step 4: Run the resolver tests**

```bash
go test ./tools/ -run TestResolveAndCheck -v
```

Expected: PASS.

- [ ] **Step 5: Write the failing test for pinned dialing**

Append to `tools/core/web_fetch_test.go`:

```go
func TestWebFetch_UsesPinnedIP(t *testing.T) {
	// If web_fetch re-resolves at dial time, a host whose DNS flipped from
	// public → private between validation and connect would reach the
	// private target. We can't easily simulate DNS rebinding in a unit test,
	// but we can assert the code path exists by checking the transport has a
	// non-default DialContext.
	wf := NewWebFetch()
	tr, ok := wf.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", wf.client.Transport)
	}
	if tr.DialContext == nil {
		t.Fatal("expected custom DialContext for SSRF IP pinning")
	}
}
```

- [ ] **Step 6: Run to verify it fails**

```bash
go test ./tools/core/ -run TestWebFetch_UsesPinnedIP -v
```

Expected: FAIL — transport is nil/default.

- [ ] **Step 7: Wire the pinned dialer into web_fetch**

In `tools/core/web_fetch.go`, update imports to include `net` and rewrite `NewWebFetch`:

```go
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
```

Replace `NewWebFetch`:

```go
func NewWebFetch() *WebFetch {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := tools.ResolveAndCheck(host)
			if err != nil {
				return nil, fmt.Errorf("SSRF blocked at dial: %w", err)
			}
			// Try each pinned IP until one connects.
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
			if err := tools.CheckSSRF(req.URL.String()); err != nil {
				return fmt.Errorf("SSRF blocked on redirect: %w", err)
			}
			return nil
		},
	}
	return &WebFetch{client: client}
}
```

- [ ] **Step 8: Run all web_fetch tests**

```bash
go test ./tools/core/ -run TestWebFetch -v
```

Expected: PASS (redirect + pinned-IP tests both pass).

- [ ] **Step 9: Run the full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add tools/ssrf.go tools/ssrf_test.go tools/core/web_fetch.go tools/core/web_fetch_test.go
git commit -m "tools: pin resolved IP through dialer to close DNS TOCTOU"
```

---

## Task 3: Expand exec metacharacter blocklist

**Why:** `exec.go` passes commands through `sh -c`. The filter at `exec.go:67` blocks `;|&&||` `` ` `` `$(` `>` `<` but misses `\n`, `\r`, glob (`*`, `?`), brace (`{`), tilde (`~`), and bare `&`/`|`. A payload like `"cat foo\nrm -rf ~"` runs two commands. Minimal fix: extend the metachar list. (Tokenizing off `sh -c` entirely is the "right" fix but breaks quoted args; tracked as a follow-up.)

**Files:**
- Modify: `tools/core/exec.go:67` (expand `shellMetacharacters`)
- Modify: `tools/core/exec_test.go` (add case per new metachar)

- [ ] **Step 1: Write the failing tests**

Append to `tools/core/exec_test.go`:

```go
func TestIsAllowed_RejectsNewlineInjection(t *testing.T) {
	if IsAllowed("cat foo\nrm -rf /tmp/x") {
		t.Fatal("newline should block chained command")
	}
}

func TestIsAllowed_RejectsCarriageReturn(t *testing.T) {
	if IsAllowed("cat foo\rrm -rf /") {
		t.Fatal("CR should block")
	}
}

func TestIsAllowed_RejectsTildeExpansion(t *testing.T) {
	if IsAllowed("ls ~/secrets") {
		t.Fatal("tilde expansion should block")
	}
}

func TestIsAllowed_RejectsBraceExpansion(t *testing.T) {
	if IsAllowed("ls {a,b}") {
		t.Fatal("brace expansion should block")
	}
}

func TestIsAllowed_RejectsSingleAmpersand(t *testing.T) {
	if IsAllowed("ls & rm foo") {
		t.Fatal("single ampersand should block")
	}
}

func TestIsAllowed_RejectsSinglePipe(t *testing.T) {
	if IsAllowed("ls | rm foo") {
		t.Fatal("single pipe should block")
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./tools/core/ -run TestIsAllowed -v
```

Expected: FAIL on the new cases.

- [ ] **Step 3: Expand the metachar list**

In `tools/core/exec.go`, replace line 67:

```go
// shellMetacharacters rejects any character that has special meaning to
// sh -c. Kept permissive on space/tab so normal args still parse.
var shellMetacharacters = []string{
	";", "|", "&", "`", "$(", "${", ">", "<",
	"\n", "\r", "*", "?", "{", "}", "~",
}
```

(Note: `&&` and `||` are covered by the single-char `&` and `|`, so we drop the two-char entries.)

- [ ] **Step 4: Run all exec tests**

```bash
go test ./tools/core/ -run TestIsAllowed -v
go test ./tools/core/ -run TestExec -v
```

Expected: PASS. If any existing test uses patterns with `*` or `?` legitimately, update it to a concrete filename — document the behavior change in the commit body.

- [ ] **Step 5: Run the full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/core/exec.go tools/core/exec_test.go
git commit -m "tools/core: block newline+glob+brace+tilde in exec metachar filter"
```

---

# Phase 2 — Hardening

## Task 4: Scrub credentials on Read-effect tool outputs

**Why:** `ScrubCredentials` exists in `tools/scrub.go` but is called nowhere. Fetched web pages, search snippets, filesystem reads, and exec stdout/stderr are returned to the LLM verbatim — any API key in those streams leaks.

**Files:**
- Create: `tools/core/external_output.go` (shared helper)
- Create: `tools/core/external_output_test.go`
- Modify: `tools/core/web_fetch.go`, `web_search.go`, `exec.go`, `filesystem.go`, `memory.go`, `knowledge_graph.go` (call the helper on string outputs)

- [ ] **Step 1: Write the failing test for the helper**

Create `tools/core/external_output_test.go`:

```go
package core

import (
	"strings"
	"testing"
)

func TestSanitizeRead_ScrubsCredentials(t *testing.T) {
	in := "Before sk-1234567890abcdefghij After"
	out := SanitizeRead(in)
	if strings.Contains(out, "sk-1234567890abcdefghij") {
		t.Fatalf("expected credential scrubbed, got %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] marker, got %q", out)
	}
}

func TestSanitizeRead_PreservesNormalText(t *testing.T) {
	in := "hello world"
	if SanitizeRead(in) != "hello world" {
		t.Fatal("plain text should pass through")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./tools/core/ -run TestSanitizeRead -v
```

Expected: FAIL — `SanitizeRead` undefined.

- [ ] **Step 3: Create the helper**

Create `tools/core/external_output.go`:

```go
package core

import "github.com/swiftdiaries/openharness/tools"

// SanitizeRead applies credential scrubbing to any string that a Read-effect
// tool is about to return to the LLM. Kept as a single chokepoint so future
// sanitization (e.g. PII redaction) extends in one place.
func SanitizeRead(s string) string {
	return tools.ScrubCredentials(s)
}

// SanitizeExternal applies scrubbing AND wraps the result in external-content
// markers. Use for content originating off-box (web, knowledge graph). For
// on-box reads (filesystem, memory, tasks, local exec), use SanitizeRead.
func SanitizeExternal(s string) string {
	return tools.WrapExternalContent(tools.ScrubCredentials(s))
}
```

- [ ] **Step 4: Run the helper test**

```bash
go test ./tools/core/ -run TestSanitizeRead -v
```

Expected: PASS.

- [ ] **Step 5: Apply `SanitizeExternal` in web_fetch**

In `tools/core/web_fetch.go`, replace the return block (lines 109-114):

```go
result := map[string]string{
	"url":     params.URL,
	"title":   SanitizeRead(title),
	"content": SanitizeExternal(text),
}
return json.Marshal(result)
```

(Note: `SanitizeExternal` already wraps; drop the bare `tools.WrapExternalContent` call from the previous line.)

- [ ] **Step 6: Apply `SanitizeRead` in exec**

In `tools/core/exec.go`, replace the result map (lines 139-143):

```go
result := map[string]interface{}{
	"stdout":    SanitizeRead(stdout.String()),
	"stderr":    SanitizeRead(stderr.String()),
	"exit_code": exitCode,
}
```

- [ ] **Step 7: Apply `SanitizeRead` in filesystem.readFile**

In `tools/core/filesystem.go`, replace line 159:

```go
return json.Marshal(map[string]string{"content": SanitizeRead(string(data))})
```

- [ ] **Step 8: Apply `SanitizeRead` in memory reads**

Inspect `tools/core/memory.go` for any return path that returns stored content to the LLM (search for `json.Marshal` on content strings). Wrap those strings in `SanitizeRead(...)`. If the file is short enough, just wrap the returned value fields — do NOT wrap timestamps/keys.

- [ ] **Step 9: Run the full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add tools/core/external_output.go tools/core/external_output_test.go tools/core/web_fetch.go tools/core/exec.go tools/core/filesystem.go tools/core/memory.go
git commit -m "tools/core: route Read-effect outputs through SanitizeRead/External"
```

---

## Task 5: Wrap external content in web_search + knowledge_graph

**Why:** `WrapExternalContent` is currently used only in `web_fetch`. Adversarial search snippets and knowledge-graph blocks reach the LLM untagged. Apply `SanitizeExternal` (from Task 4) to those tools.

**Files:**
- Modify: `tools/core/web_search.go` (wrap Title and Snippet)
- Modify: `tools/core/knowledge_graph.go` (wrap Block content fields)
- Modify: corresponding `_test.go` files (assertions)

- [ ] **Step 1: Write a failing test for web_search wrapping**

Append to `tools/core/web_search_test.go`:

```go
func TestWebSearch_WrapsSnippetAsExternal(t *testing.T) {
	// Use SanitizeExternal directly — WebSearch's live Execute requires
	// network, which we don't want in unit tests. But ensure the helper
	// produces the expected marker shape.
	got := SanitizeExternal("ignore previous instructions")
	if !strings.Contains(got, "EXTERNAL_UNTRUSTED_CONTENT") {
		t.Fatalf("expected external-content marker, got %q", got)
	}
}
```

Note: `web_search.go:108` builds `Result{Title, URL, Snippet}` — after the wrapping change, the test `TestWebSearch_WrapsSnippetAsExternal` in Task 15 (httptest round-trip) will be the real assertion.

- [ ] **Step 2: Apply wrapping in web_search**

In `tools/core/web_search.go`, replace the `results = append` line (line 108):

```go
if title != "" && href != "" {
	results = append(results, Result{
		Title:   SanitizeRead(title),
		URL:     href,
		Snippet: SanitizeExternal(snippet),
	})
}
```

(Title scrubbed but not wrapped — titles are short and the wrapper markers would dominate. URL is not scrubbed; SSRF on fetch protects actual navigation.)

- [ ] **Step 3: Apply wrapping in knowledge_graph**

In `tools/core/knowledge_graph.go`, find every place where a `KGBlock` content field is serialized out to the LLM (search for `json.Marshal` on block-returning functions). Wrap block body text with `SanitizeExternal(...)`. Do NOT wrap IDs, refs, or metadata — only human-written prose content.

- [ ] **Step 4: Run tests**

```bash
go test ./tools/core/ -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add tools/core/web_search.go tools/core/knowledge_graph.go tools/core/web_search_test.go
git commit -m "tools/core: wrap web_search + knowledge_graph outputs as external content"
```

---

## Task 6: Reject empty WorkspacePath in Register

**Why:** `core.Config.WorkspacePath = ""` silently scopes `Filesystem` and `Exec` to the process CWD — a permissive-by-default footgun. Fail loudly.

**Files:**
- Modify: `tools/core/core.go:41-44` (validate before registering)
- Modify: `tools/core/core_test.go` (new test)

- [ ] **Step 1: Write the failing test**

Append to `tools/core/core_test.go`:

```go
func TestRegister_RejectsEmptyWorkspacePath(t *testing.T) {
	r := tools.NewRegistry()
	err := Register(r, Config{WorkspacePath: ""})
	if err == nil {
		t.Fatal("expected error on empty WorkspacePath")
	}
	if !strings.Contains(err.Error(), "WorkspacePath") {
		t.Fatalf("expected WorkspacePath in error, got %v", err)
	}
}
```

Add `"strings"` to the test imports if missing.

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./tools/core/ -run TestRegister_RejectsEmptyWorkspacePath -v
```

Expected: FAIL — registration currently succeeds.

- [ ] **Step 3: Add the guard**

In `tools/core/core.go`, replace the body of `Register` (starting at line 41):

```go
func Register(r *tools.Registry, cfg Config) error {
	if cfg.WorkspacePath == "" {
		return fmt.Errorf("core: WorkspacePath must not be empty; pass an explicit path")
	}
	if err := r.Register("web", NewWebSearch()); err != nil {
		return err
	}
	// ... rest unchanged
```

Add `"fmt"` to the imports.

- [ ] **Step 4: Run test**

```bash
go test ./tools/core/ -run TestRegister_RejectsEmptyWorkspacePath -v
```

Expected: PASS.

- [ ] **Step 5: Fix existing tests that passed empty WorkspacePath**

```bash
go test ./tools/core/ -v
```

Any test that previously called `Register` with the zero-valued `Config{}` will now fail. Update each to use `Config{WorkspacePath: t.TempDir()}`. Do this surgically — run the failures and update one by one.

- [ ] **Step 6: Run the full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tools/core/core.go tools/core/core_test.go
git commit -m "tools/core: reject empty WorkspacePath in Register"
```

---

# Phase 3 — Framework Hygiene

## Task 7: Split exec into read-only and mutating definitions

**Why:** `exec` is registered with `ToolEffectNeutral` but its allowlist includes `cp`, `mv`, `mkdir`, `touch` — all mutating. ModePlan (Plan 5) filters by effect, so these commands bypass the mutate gate. Split `exec` into two definitions with different allowlists and correct effects.

**Files:**
- Modify: `tools/core/exec.go` (two Definitions; split allowlists)
- Modify: `tools/core/exec_test.go` (assert both definitions exist with correct effects)
- Modify: `tools/core/core_test.go` (D4 matrix updated: `exec` → `exec_read` + `exec_mutate`)

- [ ] **Step 1: Write the failing test**

Append to `tools/core/exec_test.go`:

```go
func TestExec_DefinitionsSplitByEffect(t *testing.T) {
	e := NewExec(t.TempDir())
	defs := e.Definitions()
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions (read + mutate), got %d", len(defs))
	}
	var seenRead, seenMutate bool
	for _, d := range defs {
		switch d.Effects {
		case tools.ToolEffectRead, tools.ToolEffectNeutral:
			seenRead = true
		case tools.ToolEffectMutate:
			seenMutate = true
		}
	}
	if !seenRead || !seenMutate {
		t.Fatalf("expected both read and mutate, got read=%v mutate=%v", seenRead, seenMutate)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./tools/core/ -run TestExec_DefinitionsSplitByEffect -v
```

Expected: FAIL.

- [ ] **Step 3: Implement the split**

In `tools/core/exec.go`, split the allowlist:

```go
// readOnlyCommands are inspection-only commands — safe for Read/Neutral effect.
var readOnlyCommands = map[string]bool{
	"ls": true, "cat": true, "grep": true, "rg": true,
	"git": true, "wc": true, "head": true, "tail": true,
	"date": true, "echo": true, "pwd": true, "find": true,
	"diff": true, "sort": true, "uniq": true, "cut": true,
	"tr": true, "jq": true,
	"which": true, "file": true, "stat": true, "tree": true,
	"du": true, "df": true, "uname": true, "whoami": true,
}

// mutatingCommands modify the workspace — require Mutate effect.
var mutatingCommands = map[string]bool{
	"mkdir": true, "cp": true, "mv": true, "touch": true,
}
```

Rewrite `Definitions()`:

```go
func (e *Exec) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "exec_read",
			Description: "Execute a read-only shell command (ls, cat, grep, git-status, etc.) in the workspace.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {"type": "string"},
					"timeout_ms": {"type": "integer"}
				},
				"required": ["command"]
			}`),
			Effects: tools.ToolEffectNeutral,
		},
		{
			Name:        "exec_mutate",
			Description: "Execute a workspace-mutating shell command (mkdir, cp, mv, touch).",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {"type": "string"},
					"timeout_ms": {"type": "integer"}
				},
				"required": ["command"]
			}`),
			Effects: tools.ToolEffectMutate,
		},
	}
}
```

Update `Execute` to route by name and enforce the correct allowlist. Update `IsAllowed` to take an effect argument or introduce `IsAllowedRead` / `IsAllowedMutate`. Replace the old `allowedCommands` reference with a helper:

```go
func allowedFor(name, command string) bool {
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) == 0 {
		return false
	}
	base := filepath.Base(parts[0])
	switch name {
	case "exec_read":
		return readOnlyCommands[base]
	case "exec_mutate":
		return mutatingCommands[base]
	}
	return false
}
```

(Route in `Execute`: after the metachar check and `MatchesDenyPattern`, dispatch via `allowedFor(name, params.Command)`.)

- [ ] **Step 4: Update the D4 matrix in core_test.go**

Replace the `exec` row with `exec_read` (Neutral) and `exec_mutate` (Mutate).

- [ ] **Step 5: Run full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/core/exec.go tools/core/exec_test.go tools/core/core_test.go
git commit -m "tools/core: split exec into read-only + mutating tools with correct Effects"
```

---

## Task 8: Tighten memory + tasks file permissions to 0600

**Why:** `memory.json` and `tasks.json` may contain PII, extracted secrets, or private notes. `0644` lets any local user read them.

**Files:**
- Modify: `tools/core/memory.go:55` (0o644 → 0o600)
- Modify: `tools/core/tasks.go:209` (0644 → 0600, also `MkdirAll` dir mode 0755 → 0700)
- Modify: `tools/core/memory_test.go`, `tasks_test.go` (assert mode)

- [ ] **Step 1: Write the failing test**

Append to `tools/core/memory_test.go`:

```go
func TestMemory_FileMode0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	m := NewMemory(path)
	// Trigger a write.
	args, _ := json.Marshal(map[string]string{"key": "k", "value": "v"})
	if _, err := m.Execute(context.Background(), "memory_store", args); err != nil {
		t.Fatalf("store: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}
```

(Add any imports: `os`, `path/filepath`, `context`, `encoding/json`. Verify the actual store-tool name in memory.go — if it's `store_file` or `memory.store`, adjust accordingly.)

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./tools/core/ -run TestMemory_FileMode0600 -v
```

Expected: FAIL — current mode is 0644.

- [ ] **Step 3: Change perms in memory.go**

Line 55 (`os.WriteFile(..., 0o644)`) → `0o600`. Also tighten the enclosing `MkdirAll` to `0700` if present.

- [ ] **Step 4: Change perms in tasks.go**

Line 209 (`0644`) → `0o600`. Directory `MkdirAll` at lines 184, 204 → `0700`.

- [ ] **Step 5: Add mirror test for tasks**

Append similar `TestTasks_FileMode0600` to `tools/core/tasks_test.go`.

- [ ] **Step 6: Run full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tools/core/memory.go tools/core/tasks.go tools/core/memory_test.go tools/core/tasks_test.go
git commit -m "tools/core: tighten memory+tasks file perms to 0600"
```

---

## Task 9: Registry name-collision guard + delete deferredActivator

**Why:** `Registry.Register` silently overwrites an existing tool with the same name — low-severity today, trivial to prevent. `deferredActivator` field + setter have zero in-repo callers; Plan 4 can add the functionality cheaply when it has a caller.

**Files:**
- Modify: `tools/registry.go` (add collision check in `Register`; delete `deferredActivator` field, setter, and the `if r.deferredActivator != nil ...` block)
- Modify: `tools/registry_test.go` (new collision test; remove deferred tests if any)

- [ ] **Step 1: Check for actual callers before deleting**

```bash
grep -rn "SetDeferredActivator\|deferredActivator" --include="*.go"
```

Expected: only `tools/registry.go` itself references these symbols. If any external caller exists, STOP and reassess — deletion is wrong.

- [ ] **Step 2: Write the failing collision test**

Append to `tools/registry_test.go`:

```go
func TestRegister_RejectsNameCollision(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("cat", fakeTool{name: "foo", effect: ToolEffectRead}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := r.Register("cat", fakeTool{name: "foo", effect: ToolEffectRead})
	if err == nil {
		t.Fatal("expected collision error")
	}
}
```

(If `fakeTool` doesn't exist in `registry_test.go`, inline a minimal stub Tool that returns one `ToolDefinition{Name: "foo", Effects: ToolEffectRead}`.)

- [ ] **Step 3: Run to verify it fails**

```bash
go test ./tools/ -run TestRegister_RejectsNameCollision -v
```

Expected: FAIL.

- [ ] **Step 4: Add the guard**

In `tools/registry.go:42-55`, inside `Register` before inserting into the name-keyed map, return an error if the name already exists:

```go
if _, dup := r.tools[def.Name]; dup {
	return fmt.Errorf("tool name collision: %q already registered", def.Name)
}
```

(Exact line depends on how tools are keyed — verify the map's name.)

- [ ] **Step 5: Delete `deferredActivator`**

Remove:
- The `deferredActivator` field at `tools/registry.go:25`.
- The setter method around `tools/registry.go:114`.
- The lookup block around `tools/registry.go:120` that checks `r.deferredActivator != nil`.

Update any comment that references Plan 4 deferred activation to instead say: "Plan 4 will add deferred activation when MCP caller exists."

- [ ] **Step 6: Run all tests**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tools/registry.go tools/registry_test.go
git commit -m "tools: reject name collisions in Registry; drop unused deferredActivator"
```

---

## Task 10: Delete unused ToolVisibility

**Why:** Field is set and read nowhere outside of its own test. Plan 4 can re-introduce if/when UIBridge needs it; speculative API surface rots.

**Files:**
- Modify: `tools/tool.go:46-55, 69` (delete enum and struct field)
- Modify: `tools/tool_test.go` (delete TestToolVisibility* tests)

- [ ] **Step 1: Verify no external references**

```bash
grep -rn "ToolVisibility\|Visibility" --include="*.go" | grep -v "_test.go"
```

Expected: only `tools/tool.go`.

- [ ] **Step 2: Delete the enum and field**

Remove lines 46-55 (the `ToolVisibility` type + constants) and line 69 (the `Visibility` field on `ToolDefinition`).

- [ ] **Step 3: Delete the tests**

Remove `TestToolVisibilityZeroValueIsDefault` and any other `Visibility` assertions from `tools/tool_test.go`.

- [ ] **Step 4: Run full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add tools/tool.go tools/tool_test.go
git commit -m "tools: drop unused ToolVisibility; Plan 4 reintroduces when needed"
```

---

## Task 11: Extract filesystem write-guard as injectable callback

**Why:** `tools/core/filesystem.go:16-31, 174-183, 241-250` encode ghostfin-specific semantics (`agentType == "predefined"`, `IDENTITY.md`/`AGENTS.md`/`SOUL.md`, `selfEvolve`) inside a framework package that claims vertical-agnosticism. Extract as a pluggable `WriteGuardFunc(absPath string) error` that verticals supply. openharness ships no default guard.

**Files:**
- Modify: `tools/core/filesystem.go` (replace the fields + `SetWriteGuard` + inline guards)
- Modify: `tools/core/filesystem_test.go` (exercise the callback)
- Modify: `tools/core/core.go` (add optional `WriteGuard` to `Config`)

- [ ] **Step 1: Write the failing test**

Replace the existing write-guard tests in `tools/core/filesystem_test.go` with:

```go
func TestFilesystem_WriteGuardBlocks(t *testing.T) {
	ws := t.TempDir()
	fs := NewFilesystem(ws)
	fs.SetWriteGuard(func(absPath string) error {
		if filepath.Base(absPath) == "BLOCKED.md" {
			return fmt.Errorf("BLOCKED.md is immutable")
		}
		return nil
	})

	args, _ := json.Marshal(map[string]string{"path": "BLOCKED.md", "content": "x"})
	_, err := fs.Execute(context.Background(), "write_file", args)
	if err == nil || !strings.Contains(err.Error(), "BLOCKED.md") {
		t.Fatalf("expected BLOCKED.md error, got %v", err)
	}

	args2, _ := json.Marshal(map[string]string{"path": "ok.md", "content": "x"})
	if _, err := fs.Execute(context.Background(), "write_file", args2); err != nil {
		t.Fatalf("unblocked write failed: %v", err)
	}
}

func TestFilesystem_NoGuardAllowsAll(t *testing.T) {
	ws := t.TempDir()
	fs := NewFilesystem(ws)
	args, _ := json.Marshal(map[string]string{"path": "anything.md", "content": "x"})
	if _, err := fs.Execute(context.Background(), "write_file", args); err != nil {
		t.Fatalf("no guard should allow write: %v", err)
	}
}
```

Delete the old `agentType == "predefined"` / `SOUL.md` tests.

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./tools/core/ -run TestFilesystem_WriteGuard -v
```

Expected: FAIL.

- [ ] **Step 3: Refactor Filesystem**

In `tools/core/filesystem.go`:

```go
// WriteGuardFunc returns an error to block a write operation. Verticals
// supply this to protect context files; openharness ships no default.
type WriteGuardFunc func(absPath string) error

type Filesystem struct {
	workspace string
	guard     WriteGuardFunc
}

func NewFilesystem(workspace string) *Filesystem {
	resolved, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		resolved = filepath.Clean(workspace)
	}
	return &Filesystem{workspace: resolved}
}

func (f *Filesystem) SetWriteGuard(fn WriteGuardFunc) {
	f.guard = fn
}
```

Replace the inline guards at `writeFile` (old lines 174-183) and `editFile` (old lines 241-250):

```go
if f.guard != nil {
	if err := f.guard(resolved); err != nil {
		return nil, err
	}
}
```

Delete `predefinedImmutableFiles`, `agentType`, `selfEvolve`, and the old `SetWriteGuard(agentType string, selfEvolve bool)` signature. Delete `DefaultWorkspace()` (Task 12 covers this — do it in this commit if convenient, or split if it cascades to other files).

- [ ] **Step 4: Wire WriteGuard through Config**

In `tools/core/core.go`, add to `Config`:

```go
// WriteGuard, if non-nil, is attached to the filesystem tool to block
// writes to specific paths (e.g. vertical-owned context files). nil is fine.
WriteGuard WriteGuardFunc
```

In `Register`, after constructing the filesystem:

```go
fs := NewFilesystem(cfg.WorkspacePath)
if cfg.WriteGuard != nil {
	fs.SetWriteGuard(cfg.WriteGuard)
}
if err := r.Register("filesystem", fs); err != nil {
	return err
}
```

- [ ] **Step 5: Run full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/core/filesystem.go tools/core/filesystem_test.go tools/core/core.go
git commit -m "tools/core: extract filesystem write-guard as injectable WriteGuardFunc"
```

---

## Task 12: Delete DefaultWorkspace() (~/.ghostfin/workspace)

**Why:** Hardcoded ghostfin path in a framework package. openharness verticals should choose their own default (ghostfin-enterprise can reintroduce in its own package).

**Files:**
- Modify: `tools/core/filesystem.go:43-47` (delete `DefaultWorkspace`)
- Grep for callers; update or remove.

- [ ] **Step 1: Find callers**

```bash
grep -rn "DefaultWorkspace\b" --include="*.go"
```

If ghostfin is a caller (it's a separate repo), that's fine — they'll see a compile error and relocate the helper to their side. Within openharness, no caller should remain.

- [ ] **Step 2: Delete the function**

Remove lines 43-47 of `tools/core/filesystem.go`.

- [ ] **Step 3: Run full suite**

```bash
go test ./tools/... ./...
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add tools/core/filesystem.go
git commit -m "tools/core: drop DefaultWorkspace (~/.ghostfin path is vertical-specific)"
```

---

## Task 13: Rename GhostFin User-Agent to openharness

**Why:** openharness-emitted HTTP requests currently advertise themselves as `GhostFin/1.0`. Embarrassing and misleading for other verticals using the library.

**Files:**
- Modify: `tools/core/web_fetch.go:65`
- Modify: `tools/core/web_search.go:66`

- [ ] **Step 1: Change both User-Agent strings**

Both files currently:

```go
req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GhostFin/1.0)")
```

Change to:

```go
req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; openharness/0.1)")
```

- [ ] **Step 2: Run full suite**

```bash
go test ./tools/... ./...
```

Expected: PASS (no test asserts on User-Agent today).

- [ ] **Step 3: Commit**

```bash
git add tools/core/web_fetch.go tools/core/web_search.go
git commit -m "tools/core: rename HTTP User-Agent from GhostFin to openharness"
```

---

# Phase 4 — Polish

## Task 14: Add tools/README.md for tool authors

**Why:** Effects taxonomy and `Register(r, cfg)` contract are only documented in godoc comments. A new contributor has no single entry point.

**Files:**
- Create: `tools/README.md`

- [ ] **Step 1: Write the README**

Create `tools/README.md` with these sections (fill with real code taken from the repo — do not abbreviate):

1. **What is this package** (2 sentences)
2. **The Tool interface** — show the `Tool` interface and `ToolDefinition` struct verbatim from `tool.go`.
3. **Effects taxonomy** — table of Read / Mutate / Neutral / Interactive with one-line explanation of each and an example tool.
4. **Registering built-ins** — show a minimal `main.go` that constructs a `Registry`, calls `core.Register(r, Config{WorkspacePath: "..."})`, and lists `r.EffectsFor("read_file")`.
5. **Authoring a new tool** — show a minimal `type MyTool struct{}` implementing `Definitions() []ToolDefinition` and `Execute(ctx, name, args) (json.RawMessage, error)`.
6. **External-content wrapping** — when to use `SanitizeRead` vs `SanitizeExternal` (from Task 4).
7. **Testing conventions** — point to `core_test.go` as the canonical D4 matrix shape.

Keep it under 250 lines. Reference actual paths (`tools/core/core.go:41`, `tools/tool.go:22`) so readers can jump.

- [ ] **Step 2: Commit**

```bash
git add tools/README.md
git commit -m "docs: add tools/README.md with Effects taxonomy + authoring guide"
```

---

## Task 15: Add httptest round-trip tests for web_fetch and web_search

**Why:** Current web tests only assert the `Definitions()` shape. DuckDuckGo HTML parsing in `web_search.go:74-130` is the piece most likely to silently rot when the upstream site changes. A fixed-fixture round-trip catches regressions without network flake.

**Files:**
- Modify: `tools/core/web_fetch_test.go` (httptest server + Execute round-trip)
- Modify: `tools/core/web_search_test.go` (fixture HTML + Execute round-trip)

- [ ] **Step 1: Write the web_fetch round-trip test**

Append to `tools/core/web_fetch_test.go`:

```go
func TestWebFetch_ExecuteRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>Hello</title></head><body><p>World</p><script>alert(1)</script></body></html>`)
	}))
	defer srv.Close()

	wf := NewWebFetch()
	args, _ := json.Marshal(map[string]string{"url": srv.URL})
	raw, err := wf.Execute(context.Background(), "web_fetch", args)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(out["title"], "Hello") {
		t.Fatalf("title: %q", out["title"])
	}
	if !strings.Contains(out["content"], "World") {
		t.Fatalf("content missing body: %q", out["content"])
	}
	if strings.Contains(out["content"], "alert(1)") {
		t.Fatalf("script content should be stripped")
	}
	if !strings.Contains(out["content"], "EXTERNAL_UNTRUSTED_CONTENT") {
		t.Fatalf("expected external-content wrapper")
	}
}
```

Note: `httptest.NewServer` binds to 127.0.0.1 — which is a private IP. For this to pass, either (a) add an opt-in SSRF bypass for `httptest`-originated URLs in tests via a hook, or (b) use a `PrivateIPSafe` flag exposed for tests. Simplest: add a package-level `var disableSSRFForTests = false` in web_fetch.go, set it to true at the top of the test, and have `CheckSSRF` short-circuit when it's set. Reset it in a `t.Cleanup`.

- [ ] **Step 2: Add the test hook in web_fetch.go**

Add at the top of `web_fetch.go`:

```go
// disableSSRFForTests opts out of SSRF checks. Set ONLY from *_test.go in
// this package; not exported.
var disableSSRFForTests bool
```

Gate both checks:

```go
if !disableSSRFForTests {
	if err := tools.CheckSSRF(params.URL); err != nil {
		...
	}
}
```

And similarly in the `CheckRedirect` + `DialContext` closures. This is test-only plumbing; the production path is unaffected.

- [ ] **Step 3: Use the hook in the test**

In the test:

```go
disableSSRFForTests = true
t.Cleanup(func() { disableSSRFForTests = false })
```

- [ ] **Step 4: Write the web_search round-trip test**

Append to `tools/core/web_search_test.go`:

```go
func TestWebSearch_ExecuteRoundTrip(t *testing.T) {
	// Minimal DuckDuckGo-shaped HTML.
	fixture := `<html><body>
		<div class="result">
		  <a class="result__a" href="https://example.com/a">Result A</a>
		  <div class="result__snippet">Snippet A text</div>
		</div>
	</body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fixture)
	}))
	defer srv.Close()

	ws := NewWebSearch()
	// Point the search URL at the fixture server. This requires adding a
	// package-private var searchBaseURL = "https://html.duckduckgo.com/html/"
	// in web_search.go, which the test overrides.
	prev := searchBaseURL
	searchBaseURL = srv.URL + "/?q="
	t.Cleanup(func() { searchBaseURL = prev })

	args, _ := json.Marshal(map[string]string{"query": "anything"})
	raw, err := ws.Execute(context.Background(), "web_search", args)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var results []map[string]string
	if err := json.Unmarshal(raw, &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if !strings.Contains(results[0]["title"], "Result A") {
		t.Fatalf("title: %q", results[0]["title"])
	}
	if !strings.Contains(results[0]["snippet"], "EXTERNAL_UNTRUSTED_CONTENT") {
		t.Fatalf("expected external-content wrapper on snippet")
	}
}
```

- [ ] **Step 5: Add the extract point in web_search.go**

Replace the hardcoded URL at line 61 with:

```go
var searchBaseURL = "https://html.duckduckgo.com/html/?q="
```

And use `searchBaseURL + url.QueryEscape(params.Query)`.

- [ ] **Step 6: Run the tests**

```bash
go test ./tools/core/ -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tools/core/web_fetch.go tools/core/web_search.go tools/core/web_fetch_test.go tools/core/web_search_test.go
git commit -m "tools/core: add httptest round-trip tests for web_fetch and web_search"
```

---

## Task 16: Fix dropped errors and shadowed err

**Why:** `tasks.go:184, 199, 204` and `filesystem.go:184` drop `os.MkdirAll` / `json.Unmarshal` errors. `knowledge_graph.go:115-132` shadows `err`. Both patterns break loudly later when someone refactors.

**Files:**
- Modify: `tools/core/tasks.go:184, 199, 204`
- Modify: `tools/core/filesystem.go:184`
- Modify: `tools/core/knowledge_graph.go:115-132`

- [ ] **Step 1: Fix tasks.go:199** (Unmarshal)

Replace:

```go
json.Unmarshal(data, &tasks)
```

With:

```go
if err := json.Unmarshal(data, &tasks); err != nil {
	return nil, fmt.Errorf("parse tasks file: %w", err)
}
```

Adjust error return types to match the enclosing function.

- [ ] **Step 2: Fix tasks.go:184 and :204** (MkdirAll)

Replace:

```go
os.MkdirAll(dir, 0700)
```

With:

```go
if err := os.MkdirAll(dir, 0700); err != nil {
	return nil, fmt.Errorf("mkdir tasks dir: %w", err)
}
```

- [ ] **Step 3: Fix filesystem.go:184** (MkdirAll)

Same pattern.

- [ ] **Step 4: Fix knowledge_graph.go:115-132** (shadowed err)

Rename the inner `err` to `perr` (or restructure so only one `err` is in scope). Example: if the code looks like:

```go
if params.Page != "" {
	page, err := store.GetPage(params.Page)
	if err != nil { return nil, err }
	blocks, err = store.SearchInPage(page, params.Query)
	...
}
```

Rewrite to:

```go
if params.Page != "" {
	page, perr := store.GetPage(params.Page)
	if perr != nil { return nil, perr }
	var perr2 error
	blocks, perr2 = store.SearchInPage(page, params.Query)
	if perr2 != nil { return nil, perr2 }
}
```

Verify the exact line numbers against the current file before editing — the review flagged lines 115-132 but the specific pattern may differ slightly.

- [ ] **Step 5: Run full suite**

```bash
go vet ./...
go test ./tools/... ./...
```

Expected: PASS, no vet warnings.

- [ ] **Step 6: Commit**

```bash
git add tools/core/tasks.go tools/core/filesystem.go tools/core/knowledge_graph.go
git commit -m "tools/core: handle dropped errors; disambiguate shadowed err"
```

---

## Task 17: Add TODO(plan-4) marker in ask_user test

**Why:** `ask_user_test.go` asserts a vacuous stub (`{"status":"waiting_for_user"}`). When Plan 4 wires real channels, the test won't update unless flagged.

**Files:**
- Modify: `tools/core/ask_user_test.go` (add comment + link to plan 4 issue)

- [ ] **Step 1: Add the marker**

Above the test body, add:

```go
// TODO(plan-4): this test asserts the stubbed "waiting_for_user" response.
// When Plan 4 wires the UIBridge channel, rewrite this test to exercise the
// real interactive flow. See docs/superpowers/plans/layer-2/*-plan-4-*.md
// (or the beads issue once filed).
```

- [ ] **Step 2: Commit**

```bash
git add tools/core/ask_user_test.go
git commit -m "tools/core: flag ask_user stub test with TODO(plan-4)"
```

---

# Wrap-up

- [ ] **Run preflight checks**

```bash
go build ./...
go vet ./...
go test ./tools/... ./...
bd preflight
```

- [ ] **File beads issues for any deferred items**

If any task above got deferred (e.g. the full exec tokenization refactor beyond Task 3), create a beads issue capturing it:

```bash
bd create --title="exec: replace sh -c with tokenized exec.Command" --description="Task 3 expanded the metachar blocklist as a minimal fix. Long-term, exec should parse args and call exec.CommandContext directly so shell injection becomes structurally impossible. Preserves quoted-arg support is the main design question." --type=task --priority=2
```

- [ ] **Push**

```bash
git pull --rebase
bd dolt push
git push
git status  # must show "up to date with origin"
```

---

## Self-Review Checklist

- **Spec coverage**: every review finding above is mapped to a task (SSRF redirect → T1; DNS TOCTOU → T2; exec newline → T3; scrub dead code → T4; wrap coverage → T5; empty workspace → T6; exec Neutral-on-mutate → T7; 0644 perms → T8; collision + deferredActivator → T9; ToolVisibility → T10; ghostfin filesystem leak → T11; DefaultWorkspace → T12; User-Agent → T13; tools/README → T14; httptest → T15; dropped/shadowed err → T16; ask_user TODO → T17). ✅
- **No placeholders**: every code step shows actual code; every command has an expected outcome; no "TBD"/"etc." ✅
- **Type consistency**: `WriteGuardFunc` is defined in T11 and used in T11 only. `SanitizeRead`/`SanitizeExternal` are defined in T4 and used in T4, T5. `ResolveAndCheck` is defined in T2 (step 3) and used in T2 (step 7). ✅

---

## Risks & Sequencing Notes

- **Task 6** (reject empty WorkspacePath) will break any existing test that called `Register(r, Config{})`. Step 5 of T6 handles this surgically. Don't skip it.
- **Task 7** (split `exec`) renames `exec` → `exec_read` + `exec_mutate`. Any downstream caller (ghostfin-enterprise, future vertical) that hard-references `"exec"` will break. Document in the commit body.
- **Task 11** deletes `DefaultWorkspace` from the framework — ghostfin-enterprise must reintroduce it on their side. File a beads issue against ghostfin-enterprise if that repo is sharing CI.
- **Phase order matters**: Phase 1 must land before any production/shared deployment that exposes these tools behind an LLM with network access.
