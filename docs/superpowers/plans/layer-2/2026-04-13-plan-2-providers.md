# Layer 2 Plan 2: Providers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move `internal/providers/{types,models,registry,openai_compat,openrouter}.go` from ghostfin into `openharness/providers/` verbatim (with `sqliteonly` build tags stripped and the `keychain` dependency dropped), then add a new `openharness/providers/anthropic.go` using the official `github.com/anthropics/anthropic-sdk-go` with real SSE streaming, prompt-caching support, tool-call handling, and cache-token accounting.

**Architecture:** The existing ghostfin `Provider` interface (`Chat`, `ChatStream`, `DefaultModel`, `Name`) and supporting types (`ChatRequest`, `ChatResponse`, `StreamChunk`, `Usage`, `Message`, `ToolCall`, `ToolDefinition`) are preserved identically. `OpenAICompatProvider`, `Registry`, and OpenRouter helpers move verbatim — no behavior changes. `anthropic.go` is a new implementation of the same `Provider` interface built on the Anthropic SDK's `Messages.New` / `Messages.NewStreaming` APIs; it closes a latent gap where `Usage.CacheCreationTokens` / `Usage.CacheReadTokens` fields exist but no previous provider populated them.

**Tech Stack:** Go 1.25, `github.com/anthropics/anthropic-sdk-go` (new dep), `go.opentelemetry.io/otel/trace` (already in openharness go.mod via Layer 1), stdlib `net/http/httptest` for tests. No API keys required in CI.

**Spec:** `docs/superpowers/specs/2026-04-13-openharness-layer-2-agent-primitives-design.md` (Sections "providers/anthropic.go — native SDK provider", Decision Q3 in key-decisions table).

**Execution context:** `docs/superpowers/plans/layer-2/2026-04-13-execution-order.md` (Wave 1B — originally scoped to run in parallel with Plan 1, now runs solo after Plan 1 has already merged; no file overlap with Plan 1's `agent/cost/sessions/edition/config/` work).

---

## Pipeline Tracking

| Field | Value |
|-------|-------|
| Epic | `openharness-ssx` |
| Status | complete (PR #2) |
| Worktree | `.worktrees/plan-2-providers` (branch: `feat/layer-2-plan-2-providers`) |
| PR | https://github.com/swiftdiaries/openharness/pull/2 |
| Fast-follow | `openharness-1cu` — `ResolveAPIKeyWithStore` + `secret:<name>` scheme (landed in the same PR) |

| Task | Beads ID | Status |
|------|----------|--------|
| Task 1: Worktree sanity + add Anthropic SDK dep | `openharness-qwi` | closed |
| Task 2: Move `providers/types.go` | `openharness-by6` | closed |
| Task 3: Move `providers/models.go` | `openharness-c31` | closed |
| Task 4: Move `providers/registry.go` + test | `openharness-oe8` | closed |
| Task 5: Move `providers/openai_compat.go` + test | `openharness-xyc` | closed |
| Task 6: Move `providers/openrouter.go` + test (drop keychain dep) | `openharness-8zj` | closed |
| Task 7: `providers/anthropic.go` skeleton + constructor test | `openharness-x9g` | closed |
| Task 8: `anthropic.go` non-streaming `Chat` (text response) | `openharness-78y` | closed |
| Task 9: `anthropic.go` streaming `ChatStream` (text deltas) | `openharness-bl0` | closed |
| Task 10: `anthropic.go` tool-call reassembly | `openharness-32t` | closed |
| Task 11: `anthropic.go` prompt caching + cache-token accounting | `openharness-2v0` | closed |
| Task 12: `anthropic.go` error handling + retry semantics | `openharness-d31` | closed |
| Task 13: Final verification, lint, push | `openharness-bup` | closed |
| Fast-follow: `ResolveAPIKeyWithStore` + `secret:` scheme | `openharness-1cu` | closed |

**Before Task 1:** create a beads epic and the 13 tasks above. Claim each task before starting it and close it after its commit lands.

```bash
bd create --type=epic --title="Layer 2 Plan 2: providers (OpenAI-compat move + native Anthropic)" \
  --description="Move providers verbatim from ghostfin, strip sqliteonly, drop keychain dep; add native Anthropic SDK provider with SSE streaming, tool-call reassembly, and prompt caching." \
  --priority=2
# Record the returned epic ID in the table above, then create 13 child tasks with
# --depends-on=<epic-id> so bd ready surfaces them in order.
```

---

## File Inventory

### Files to copy from ghostfin (strip `//go:build sqliteonly` + `// +build sqliteonly` lines, otherwise verbatim)

| Source | Destination | Notes |
|--------|-------------|-------|
| `ghostfin/desktop/internal/providers/types.go` | `openharness/providers/types.go` | Strip sqliteonly tag. No other edits. |
| `ghostfin/desktop/internal/providers/models.go` | `openharness/providers/models.go` | Strip sqliteonly tag. No other edits. |
| `ghostfin/desktop/internal/providers/registry.go` | `openharness/providers/registry.go` | Strip sqliteonly tag. No other edits. |
| `ghostfin/desktop/internal/providers/registry_test.go` | `openharness/providers/registry_test.go` | Strip sqliteonly tag. No other edits. |
| `ghostfin/desktop/internal/providers/openai_compat.go` | `openharness/providers/openai_compat.go` | Strip sqliteonly tag. Otel imports stay (openharness already requires `go.opentelemetry.io/otel`). |
| `ghostfin/desktop/internal/providers/openai_compat_test.go` | `openharness/providers/openai_compat_test.go` | Strip sqliteonly tag. No other edits. |
| `ghostfin/desktop/internal/providers/openrouter.go` | `openharness/providers/openrouter.go` | Strip sqliteonly tag **and** drop `keychain:` scheme from `ResolveAPIKey` (see Task 6). |
| `ghostfin/desktop/internal/providers/openrouter_test.go` | `openharness/providers/openrouter_test.go` | Strip sqliteonly tag and remove any `keychain:` test cases. |

### Files to create new

| File | Responsibility |
|------|----------------|
| `openharness/providers/anthropic.go` | Native Anthropic `Provider` implementation: `Chat`, `ChatStream`, `DefaultModel`, `Name`. Wraps `anthropic.NewClient()` + `Messages.New` / `Messages.NewStreaming`. Honors `cache_control` on system blocks, populates `Usage.CacheCreationTokens` / `CacheReadTokens`. |
| `openharness/providers/anthropic_test.go` | Table-driven tests using `httptest.NewServer` returning canned `application/json` responses (for `Chat`) and canned `text/event-stream` bodies (for `ChatStream`). No real API calls. |
| `openharness/providers/anthropic_sse_helper_test.go` | Internal test helper that writes a sequence of SSE events (`message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop`) to an `http.ResponseWriter`. Used by streaming tests. |

### Files modified

| File | Change |
|------|--------|
| `openharness/go.mod` | Add `require github.com/anthropics/anthropic-sdk-go <latest>` (run `go get github.com/anthropics/anthropic-sdk-go@latest`). |
| `openharness/go.sum` | Updated by `go mod tidy`. |

### Files NOT moved

- `internal/providers/openrouter.go`'s `keychain:` scheme — dropped. The `ResolveAPIKey` function in openharness accepts `$ENV_VAR`, literal key, or empty-string-plus-fallback-env. Keychain / SecretStore integration is tracked as a fast-follow issue (file one at end of Plan 2).
- `internal/providers/openrouter.go`'s `VerifyAPIKey` network call — keep as-is but unchanged; ghostfin uses it at startup and Plan 7 will rewire imports.

---

## Pre-flight

Run from `.worktrees/plan-2-providers` (already created by Wave 1B setup):

```bash
cd .worktrees/plan-2-providers
git status                       # should be clean, on branch feat/layer-2-plan-2-providers
go test ./... 2>&1 | tail -20    # baseline: all existing openharness packages green
```

If baseline fails, stop and file an issue — do not proceed.

---

## Task 1: Worktree sanity + add Anthropic SDK dep

**Files:**
- Modify: `openharness/go.mod`, `openharness/go.sum`

- [ ] **Step 1: Confirm worktree state**

```bash
cd .worktrees/plan-2-providers
git branch --show-current        # expected: feat/layer-2-plan-2-providers
git status                       # expected: clean
go test ./... 2>&1 | tail -5     # expected: all existing packages passing
```

- [ ] **Step 2: Add Anthropic SDK as a dependency**

```bash
go get github.com/anthropics/anthropic-sdk-go@latest
go mod tidy
```

Expected: `go.mod` gains a `require github.com/anthropics/anthropic-sdk-go vX.Y.Z` line. `go.sum` updates.

- [ ] **Step 3: Verify compile still clean**

```bash
go build ./...
```

Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(providers): add anthropic-sdk-go dependency"
```

---

## Task 2: Move `providers/types.go`

**Files:**
- Create: `openharness/providers/types.go`
- Source: `../ghostfin/desktop/internal/providers/types.go`

- [ ] **Step 1: Copy the file**

```bash
mkdir -p providers
cp ../../../ghostfin/desktop/internal/providers/types.go providers/types.go
```

(Adjust the `../../../ghostfin` path to match the actual relative location of ghostfin on your machine; from a worktree the typical path is `../../../ghostfin/desktop/internal/providers/types.go`.)

- [ ] **Step 2: Strip the build tag**

Edit `providers/types.go`: delete the first two lines (`//go:build sqliteonly` and the blank line immediately after). Verify the file still starts with:

```go
package providers

import (
	"context"
	"encoding/json"
)
```

- [ ] **Step 3: Compile-check**

```bash
go build ./providers/...
```

Expected: `# github.com/swiftdiaries/openharness/providers — no Go files in test path` OK, or a compile error if we forgot anything. At this point, since only `types.go` is present, `go build` should succeed with no errors.

- [ ] **Step 4: Commit**

```bash
git add providers/types.go
git commit -m "feat(providers): extract Provider interface and shared types from ghostfin"
```

---

## Task 3: Move `providers/models.go`

**Files:**
- Create: `openharness/providers/models.go`
- Source: `../ghostfin/desktop/internal/providers/models.go`

- [ ] **Step 1: Copy the file**

```bash
cp ../../../ghostfin/desktop/internal/providers/models.go providers/models.go
```

- [ ] **Step 2: Strip the build tag**

Delete the `//go:build sqliteonly` line (and its trailing blank line). File should start with:

```go
package providers

// ModelContextWindows maps model identifiers to their context window size in tokens.
var ModelContextWindows = map[string]int{
```

- [ ] **Step 3: Compile-check**

```bash
go build ./providers/...
```

- [ ] **Step 4: Commit**

```bash
git add providers/models.go
git commit -m "feat(providers): extract ModelContextWindows lookup from ghostfin"
```

---

## Task 4: Move `providers/registry.go` + test

**Files:**
- Create: `openharness/providers/registry.go`
- Create: `openharness/providers/registry_test.go`

- [ ] **Step 1: Copy both files**

```bash
cp ../../../ghostfin/desktop/internal/providers/registry.go providers/registry.go
cp ../../../ghostfin/desktop/internal/providers/registry_test.go providers/registry_test.go
```

- [ ] **Step 2: Strip the build tags**

Delete the `//go:build sqliteonly` header from both files. Each should now start with `package providers`.

- [ ] **Step 3: Run the registry tests**

```bash
go test ./providers/... -run TestRegistry -v
```

Expected: all registry tests pass.

- [ ] **Step 4: Commit**

```bash
git add providers/registry.go providers/registry_test.go
git commit -m "feat(providers): extract Registry with Register/Get/List/Close from ghostfin"
```

---

## Task 5: Move `providers/openai_compat.go` + test

**Files:**
- Create: `openharness/providers/openai_compat.go`
- Create: `openharness/providers/openai_compat_test.go`

- [ ] **Step 1: Copy both files**

```bash
cp ../../../ghostfin/desktop/internal/providers/openai_compat.go providers/openai_compat.go
cp ../../../ghostfin/desktop/internal/providers/openai_compat_test.go providers/openai_compat_test.go
```

- [ ] **Step 2: Strip build tags from both**

Delete the `//go:build sqliteonly` header from both files.

- [ ] **Step 3: Fix the meta-test file path reference**

`openai_compat_test.go`'s `TestChat_NoProviderSpan` reads `openai_compat.go` by relative path and checks for a ghostfin telemetry import. The path is fine (still `openai_compat.go` in the same package directory), but the import string it greps for must be updated. Change:

```go
if strings.Contains(content, `"github.com/swiftdiaries/ghostfin/desktop/internal/telemetry"`) {
```

to:

```go
// openharness has no telemetry package; this import should never appear.
if strings.Contains(content, `/internal/telemetry"`) {
```

Leave the `telemetry.Tracer(` check as-is (the regex is generic enough).

- [ ] **Step 4: Run the openai_compat tests**

```bash
go test ./providers/... -run TestOpenAICompat -v
go test ./providers/... -run TestParseModelString -v
go test ./providers/... -run TestChat_NoProviderSpan -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add providers/openai_compat.go providers/openai_compat_test.go
git commit -m "feat(providers): extract OpenAICompatProvider + retry/parse tests from ghostfin"
```

---

## Task 6: Move `providers/openrouter.go` + test (drop `keychain:` scheme)

**Files:**
- Create: `openharness/providers/openrouter.go`
- Create: `openharness/providers/openrouter_test.go`

- [ ] **Step 1: Copy both files**

```bash
cp ../../../ghostfin/desktop/internal/providers/openrouter.go providers/openrouter.go
cp ../../../ghostfin/desktop/internal/providers/openrouter_test.go providers/openrouter_test.go
```

- [ ] **Step 2: Strip build tags from both**

Delete the `//go:build sqliteonly` header from both files.

- [ ] **Step 3: Remove the `keychain` dependency from `openrouter.go`**

Delete the import line `"github.com/swiftdiaries/ghostfin/desktop/internal/keychain"`.

Delete the `keychain:` branch from `ResolveAPIKey`. The new function body is:

```go
// ResolveAPIKey resolves an API key value.
//   - "$ENV_VAR" → reads from environment
//   - ""         → falls back to os.Getenv(fallbackEnv)
//   - anything else → returned as-is (literal key)
//
// NOTE: the ghostfin-era "keychain:<account>" scheme is not supported here;
// secret-store integration is tracked as a fast-follow (see TODO at end of
// Plan 2). In the interim, vertical apps should pass resolved literal keys
// or $ENV_VAR references at registration time.
func ResolveAPIKey(key, fallbackEnv string) string {
	if key == "" {
		return os.Getenv(fallbackEnv)
	}
	if strings.HasPrefix(key, "$") {
		return os.Getenv(key[1:])
	}
	return key
}
```

Remove the now-unused `log/slog` import if `ResolveAPIKey` was the only user of it — check with `go build ./providers/...` and remove imports flagged as unused.

- [ ] **Step 4: Remove any `keychain:` test cases from `openrouter_test.go`**

Open `providers/openrouter_test.go`. Find any test cases that exercise the `keychain:` scheme and delete them. If an entire test function (e.g., `TestResolveAPIKey_Keychain`) is now empty or references `keychain`, delete the whole function. Keep the env-var, empty-string, and literal cases.

- [ ] **Step 5: Run the openrouter tests**

```bash
go test ./providers/... -run TestResolveAPIKey -v
go test ./providers/... -run TestCallOpenRouter -v
```

Expected: all remaining tests pass, no references to `keychain` anywhere.

- [ ] **Step 6: Full providers test sweep**

```bash
go test ./providers/... -v
```

Expected: every provider test in the package now passes (types, models, registry, openai_compat, openrouter). No anthropic tests yet.

- [ ] **Step 7: Commit**

```bash
git add providers/openrouter.go providers/openrouter_test.go
git commit -m "feat(providers): extract OpenRouter helpers; drop keychain dep"
```

---

## Task 7: `providers/anthropic.go` skeleton + constructor test (TDD)

**Files:**
- Create: `openharness/providers/anthropic.go`
- Create: `openharness/providers/anthropic_test.go`

- [ ] **Step 1: Write the failing test first**

Create `providers/anthropic_test.go`:

```go
package providers

import "testing"

func TestAnthropicProvider_NameAndDefaultModel(t *testing.T) {
	p := NewAnthropicProvider("anthropic", "test-key", "claude-opus-4-6")
	if got := p.Name(); got != "anthropic" {
		t.Errorf("Name() = %q, want %q", got, "anthropic")
	}
	if got := p.DefaultModel(); got != "claude-opus-4-6" {
		t.Errorf("DefaultModel() = %q, want %q", got, "claude-opus-4-6")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./providers/... -run TestAnthropicProvider_NameAndDefaultModel -v
```

Expected: FAIL — `undefined: NewAnthropicProvider`.

- [ ] **Step 3: Write the minimal skeleton**

Create `providers/anthropic.go`:

```go
package providers

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// DefaultAnthropicModel is the fallback model used when a caller does not specify one.
const DefaultAnthropicModel = "claude-opus-4-6"

// AnthropicProvider implements Provider using the official Anthropic Go SDK.
// It supports real SSE streaming, tool-use reassembly, and prompt caching.
type AnthropicProvider struct {
	name         string
	apiKey       string
	defaultModel string
	client       anthropic.Client
}

// NewAnthropicProvider constructs a provider backed by the Anthropic SDK.
// If defaultModel is empty, DefaultAnthropicModel is used.
func NewAnthropicProvider(name, apiKey, defaultModel string) *AnthropicProvider {
	if defaultModel == "" {
		defaultModel = DefaultAnthropicModel
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicProvider{
		name:         name,
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       client,
	}
}

// newAnthropicProviderWithBaseURL is the test-only constructor that points the
// SDK at an httptest server. Not exported.
func newAnthropicProviderWithBaseURL(name, apiKey, defaultModel, baseURL string) *AnthropicProvider {
	if defaultModel == "" {
		defaultModel = DefaultAnthropicModel
	}
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	return &AnthropicProvider{
		name:         name,
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       client,
	}
}

func (p *AnthropicProvider) Name() string         { return p.name }
func (p *AnthropicProvider) DefaultModel() string { return p.defaultModel }

// Chat and ChatStream are implemented in later tasks.
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return nil, fmt.Errorf("anthropic: Chat not yet implemented")
}

func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	return nil, fmt.Errorf("anthropic: ChatStream not yet implemented")
}
```

**Executor note:** if the Anthropic SDK's client value is not named exactly `anthropic.Client` or its option package path differs, use `go doc github.com/anthropics/anthropic-sdk-go` and `go doc github.com/anthropics/anthropic-sdk-go/option` to find the correct symbols. Do **not** guess — fetch the actual types. You can also consult `mcp__plugin_context7_context7__query-docs` with library id `/anthropics/anthropic-sdk-go`.

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./providers/... -run TestAnthropicProvider_NameAndDefaultModel -v
```

Expected: PASS.

- [ ] **Step 5: Verify the whole package still compiles**

```bash
go build ./providers/...
go vet ./providers/...
```

- [ ] **Step 6: Commit**

```bash
git add providers/anthropic.go providers/anthropic_test.go
git commit -m "feat(providers): add AnthropicProvider skeleton (constructor, Name, DefaultModel)"
```

---

## Task 8: `anthropic.go` non-streaming `Chat` (text response)

**Files:**
- Modify: `openharness/providers/anthropic.go`
- Modify: `openharness/providers/anthropic_test.go`

- [ ] **Step 1: Write the failing test**

Append to `providers/anthropic_test.go`:

```go
func TestAnthropicProvider_Chat_Text(t *testing.T) {
	// Canned non-streaming response body in the Anthropic wire format.
	body := `{
		"id": "msg_test_123",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-4-6",
		"content": [{"type": "text", "text": "Hello from mock Claude"}],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 12, "output_tokens": 7}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" && r.Header.Get("X-Api-Key") == "" {
			t.Errorf("expected x-api-key header set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := newAnthropicProviderWithBaseURL("anthropic", "test-key", "claude-opus-4-6", srv.URL)

	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello from mock Claude" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello from mock Claude")
	}
	if resp.ID != "msg_test_123" {
		t.Errorf("ID = %q, want %q", resp.ID, "msg_test_123")
	}
	if resp.Usage == nil || resp.Usage.PromptTokens != 12 || resp.Usage.CompletionTokens != 7 {
		t.Errorf("Usage = %+v, want prompt=12 completion=7", resp.Usage)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "end_turn")
	}
}
```

Add imports `"context"`, `"net/http"`, `"net/http/httptest"` to the test file imports section (leave `testing` as-is).

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./providers/... -run TestAnthropicProvider_Chat_Text -v
```

Expected: FAIL — `Chat not yet implemented`.

- [ ] **Step 3: Implement `Chat`**

Replace the stub `Chat` in `providers/anthropic.go` with:

```go
// Chat sends a non-streaming Messages request and returns a ChatResponse.
// System prompts are drawn from any message with role "system" (concatenated
// with blank-line separators) and passed via MessageNewParams.System.
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	params, err := p.buildMessageParams(req)
	if err != nil {
		return nil, err
	}

	msg, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic chat: %w", err)
	}

	return p.messageToChatResponse(msg), nil
}

// buildMessageParams translates a ChatRequest into anthropic.MessageNewParams.
// Messages with role == "system" are hoisted into params.System; all others
// become MessageParam entries (user or assistant).
func (p *AnthropicProvider) buildMessageParams(req ChatRequest) (anthropic.MessageNewParams, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	// Strip any routing prefix like "openrouter/anthropic/"
	_, model = ParseModelString(p.name, model)

	var systemText string
	userOrAssistant := make([]anthropic.MessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		text, err := textFromRawContent(m.Content)
		if err != nil {
			return anthropic.MessageNewParams{}, err
		}
		switch m.Role {
		case "system":
			if systemText != "" {
				systemText += "\n\n"
			}
			systemText += text
		case "user":
			userOrAssistant = append(userOrAssistant, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		case "assistant":
			userOrAssistant = append(userOrAssistant, anthropic.NewAssistantMessage(anthropic.NewTextBlock(text)))
		default:
			return anthropic.MessageNewParams{}, fmt.Errorf("anthropic: unsupported message role %q", m.Role)
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 4096,
		Messages:  userOrAssistant,
	}
	if systemText != "" {
		params.System = []anthropic.TextBlockParam{{Text: systemText}}
	}
	return params, nil
}

// textFromRawContent unmarshals a Message.Content json.RawMessage into a
// plain string. If the RawMessage is already a JSON string, we use it
// directly; otherwise we fall back to the raw bytes.
func textFromRawContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	return string(raw), nil
}

// messageToChatResponse translates an anthropic.Message into our ChatResponse shape.
func (p *AnthropicProvider) messageToChatResponse(msg *anthropic.Message) *ChatResponse {
	resp := &ChatResponse{
		ID:           msg.ID,
		FinishReason: string(msg.StopReason),
	}

	var textParts []string
	for _, block := range msg.Content {
		switch v := block.AsAny().(type) {
		case anthropic.TextBlock:
			textParts = append(textParts, v.Text)
		case anthropic.ToolUseBlock:
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:   v.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      v.Name,
					Arguments: string(v.JSON.Input.Raw()),
				},
			})
		}
	}
	if len(textParts) > 0 {
		resp.Content = strings.Join(textParts, "")
	}

	resp.Usage = &Usage{
		PromptTokens:        int(msg.Usage.InputTokens),
		CompletionTokens:    int(msg.Usage.OutputTokens),
		TotalTokens:         int(msg.Usage.InputTokens + msg.Usage.OutputTokens),
		CacheCreationTokens: int(msg.Usage.CacheCreationInputTokens),
		CacheReadTokens:     int(msg.Usage.CacheReadInputTokens),
	}
	return resp
}
```

Add `"encoding/json"` and `"strings"` to the imports of `anthropic.go`.

**Executor note:** the exact field names on `anthropic.Message`, `anthropic.Usage`, `anthropic.ToolUseBlock`, and the `AsAny()` switch cases come from the SDK. If the names above are wrong for the pinned version, run `go doc github.com/anthropics/anthropic-sdk-go Message` and adjust. The semantics are stable across versions: there's always a content block iteration, input/output token counters, and cache-creation / cache-read counters on Usage.

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./providers/... -run TestAnthropicProvider_Chat_Text -v
```

Expected: PASS.

- [ ] **Step 5: Run the whole providers suite**

```bash
go test ./providers/... -v
```

Expected: every test green.

- [ ] **Step 6: Commit**

```bash
git add providers/anthropic.go providers/anthropic_test.go
git commit -m "feat(providers): implement AnthropicProvider.Chat (non-streaming text)"
```

---

## Task 9: `anthropic.go` streaming `ChatStream` (text deltas)

**Files:**
- Modify: `openharness/providers/anthropic.go`
- Modify: `openharness/providers/anthropic_test.go`
- Create: `openharness/providers/anthropic_sse_helper_test.go`

- [ ] **Step 1: Create the SSE helper**

Create `providers/anthropic_sse_helper_test.go`:

```go
package providers

import (
	"fmt"
	"io"
)

// writeSSEEvent writes a single Server-Sent Event to w.
// Format: "event: <name>\ndata: <json>\n\n"
func writeSSEEvent(w io.Writer, event, jsonData string) error {
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	return err
}

// writeAnthropicTextStream writes a minimal valid Anthropic streaming
// response consisting of a single text-delta content block.
// The pieces slice is joined into one assistant message.
func writeAnthropicTextStream(w io.Writer, pieces []string, inputTokens, outputTokens int) error {
	if err := writeSSEEvent(w, "message_start", fmt.Sprintf(
		`{"type":"message_start","message":{"id":"msg_stream_1","type":"message","role":"assistant","model":"claude-opus-4-6","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":%d,"output_tokens":0}}}`,
		inputTokens,
	)); err != nil {
		return err
	}
	if err := writeSSEEvent(w, "content_block_start",
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`); err != nil {
		return err
	}
	for _, piece := range pieces {
		payload := fmt.Sprintf(
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}`,
			piece,
		)
		if err := writeSSEEvent(w, "content_block_delta", payload); err != nil {
			return err
		}
	}
	if err := writeSSEEvent(w, "content_block_stop",
		`{"type":"content_block_stop","index":0}`); err != nil {
		return err
	}
	if err := writeSSEEvent(w, "message_delta", fmt.Sprintf(
		`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":%d}}`,
		outputTokens,
	)); err != nil {
		return err
	}
	return writeSSEEvent(w, "message_stop", `{"type":"message_stop"}`)
}
```

- [ ] **Step 2: Write the failing test**

Append to `providers/anthropic_test.go`:

```go
func TestAnthropicProvider_ChatStream_Text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		if err := writeAnthropicTextStream(w, []string{"Hello ", "from ", "mock ", "Claude"}, 10, 4); err != nil {
			t.Fatalf("write SSE: %v", err)
		}
	}))
	defer srv.Close()

	p := newAnthropicProviderWithBaseURL("anthropic", "test-key", "claude-opus-4-6", srv.URL)

	var received []string
	resp, err := p.ChatStream(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
	}, func(chunk StreamChunk) {
		if chunk.Content != "" {
			received = append(received, chunk.Content)
		}
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	joined := strings.Join(received, "")
	if joined != "Hello from mock Claude" {
		t.Errorf("chunks joined = %q, want %q", joined, "Hello from mock Claude")
	}
	if resp.Content != "Hello from mock Claude" {
		t.Errorf("final content = %q, want %q", resp.Content, "Hello from mock Claude")
	}
	if resp.Usage == nil || resp.Usage.PromptTokens != 10 || resp.Usage.CompletionTokens != 4 {
		t.Errorf("Usage = %+v, want prompt=10 completion=4", resp.Usage)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "end_turn")
	}
}
```

Add `"strings"` to the test file imports if not already present.

- [ ] **Step 3: Run the test to verify it fails**

```bash
go test ./providers/... -run TestAnthropicProvider_ChatStream_Text -v
```

Expected: FAIL — `ChatStream not yet implemented`.

- [ ] **Step 4: Implement `ChatStream`**

Replace the stub `ChatStream` in `providers/anthropic.go` with:

```go
// ChatStream sends a streaming Messages request. Each text delta is forwarded
// via onChunk; tool-use blocks are accumulated server-side and surfaced in the
// returned ChatResponse's ToolCalls. The final chunk (Done:true) is emitted
// once the message stream completes.
func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	params, err := p.buildMessageParams(req)
	if err != nil {
		return nil, err
	}

	stream := p.client.Messages.NewStreaming(ctx, params)

	var (
		accumulated anthropic.Message
	)
	for stream.Next() {
		event := stream.Current()
		// Accumulate into the running Message so we can assemble the final response.
		if err := accumulated.Accumulate(event); err != nil {
			return nil, fmt.Errorf("anthropic stream accumulate: %w", err)
		}

		// Forward text deltas to the caller as they arrive.
		switch e := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			if delta, ok := e.Delta.AsAny().(anthropic.TextDelta); ok {
				if onChunk != nil && delta.Text != "" {
					onChunk(StreamChunk{Content: delta.Text})
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("anthropic stream: %w", err)
	}

	if onChunk != nil {
		onChunk(StreamChunk{Done: true})
	}
	return p.messageToChatResponse(&accumulated), nil
}
```

**Executor note:** the Anthropic SDK's streaming API exposes an `Accumulate(event)` method on `Message` that reassembles content blocks + usage from deltas. If your pinned version exposes a different assembler (e.g., `Message.JSON.Accumulate` or a free function `anthropic.AccumulateStreamEvent`), use that instead — the test will tell you which shape is right.

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test ./providers/... -run TestAnthropicProvider_ChatStream_Text -v
```

Expected: PASS.

- [ ] **Step 6: Run the whole providers suite**

```bash
go test ./providers/... -v
```

- [ ] **Step 7: Commit**

```bash
git add providers/anthropic.go providers/anthropic_test.go providers/anthropic_sse_helper_test.go
git commit -m "feat(providers): implement AnthropicProvider.ChatStream with text delta forwarding"
```

---

## Task 10: `anthropic.go` tool-call reassembly

**Files:**
- Modify: `openharness/providers/anthropic.go`
- Modify: `openharness/providers/anthropic_test.go`

- [ ] **Step 1: Write the failing test**

Append to `providers/anthropic_test.go`:

```go
func TestAnthropicProvider_Chat_ToolCall(t *testing.T) {
	body := `{
		"id": "msg_tool_1",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-4-6",
		"content": [
			{"type": "text", "text": "I'll check the weather."},
			{"type": "tool_use", "id": "toolu_abc", "name": "get_weather",
			 "input": {"location": "San Francisco, CA"}}
		],
		"stop_reason": "tool_use",
		"usage": {"input_tokens": 20, "output_tokens": 15}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	p := newAnthropicProviderWithBaseURL("anthropic", "test-key", "", srv.URL)

	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "weather?")},
		Tools: []ToolDefinition{{
			Type: "function",
			Function: ToolFunctionSchema{
				Name:        "get_weather",
				Description: "Get current weather",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
			},
		}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls count = %d, want 1", len(resp.ToolCalls))
	}
	call := resp.ToolCalls[0]
	if call.ID != "toolu_abc" || call.Function.Name != "get_weather" {
		t.Errorf("ToolCall = %+v", call)
	}
	if !strings.Contains(call.Function.Arguments, "San Francisco") {
		t.Errorf("Arguments = %q, want to contain 'San Francisco'", call.Function.Arguments)
	}
	if resp.FinishReason != "tool_use" {
		t.Errorf("FinishReason = %q, want tool_use", resp.FinishReason)
	}
}
```

Add `"encoding/json"` to the test file imports if not already present.

- [ ] **Step 2: Run the test to verify it fails or passes partially**

```bash
go test ./providers/... -run TestAnthropicProvider_Chat_ToolCall -v
```

Expected: depending on how complete Task 8 was, this may already pass (because `messageToChatResponse` already switches on `ToolUseBlock`). If so, skip ahead and still commit the new test. If it fails, the failure will point at which piece of `buildMessageParams` or `messageToChatResponse` needs tools wiring.

- [ ] **Step 3: Wire tools into request building**

Extend `buildMessageParams` so callers' `ChatRequest.Tools` are translated into `anthropic.ToolUnionParam` entries on `MessageNewParams.Tools`. Add just before the `return params, nil` line:

```go
	if len(req.Tools) > 0 {
		anthTools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for _, td := range req.Tools {
			var props map[string]any
			if len(td.Function.Parameters) > 0 {
				if err := json.Unmarshal(td.Function.Parameters, &props); err != nil {
					return anthropic.MessageNewParams{}, fmt.Errorf("anthropic: tool %q parameters: %w", td.Function.Name, err)
				}
			}
			tp := anthropic.ToolParam{
				Name:        td.Function.Name,
				Description: anthropic.String(td.Function.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: props["properties"],
				},
			}
			anthTools = append(anthTools, anthropic.ToolUnionParam{OfTool: &tp})
		}
		params.Tools = anthTools
	}
```

**Executor note:** the exact field on `ToolInputSchemaParam` for properties may be `Properties map[string]any` or `Properties any` — read `go doc` to confirm, then adjust. The test is the source of truth.

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./providers/... -run TestAnthropicProvider_Chat_ToolCall -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add providers/anthropic.go providers/anthropic_test.go
git commit -m "feat(providers): wire Anthropic tool-call reassembly + tool request translation"
```

---

## Task 11: `anthropic.go` prompt caching + cache-token accounting

**Files:**
- Modify: `openharness/providers/anthropic.go`
- Modify: `openharness/providers/anthropic_test.go`

- [ ] **Step 1: Write the failing test**

Append to `providers/anthropic_test.go`:

```go
func TestAnthropicProvider_Chat_CacheTokens(t *testing.T) {
	body := `{
		"id": "msg_cache_1",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-4-6",
		"content": [{"type": "text", "text": "cached response"}],
		"stop_reason": "end_turn",
		"usage": {
			"input_tokens": 4,
			"output_tokens": 3,
			"cache_creation_input_tokens": 1024,
			"cache_read_input_tokens": 2048
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request body contains a cache_control marker on the system block.
		buf, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(buf), `"cache_control"`) {
			t.Errorf("request body missing cache_control marker: %s", buf)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	p := newAnthropicProviderWithBaseURL("anthropic", "test-key", "", srv.URL)

	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{
			NewTextMessage("system", "long stable system prompt..."),
			NewTextMessage("user", "hi"),
		},
		Options: map[string]any{"prompt_cache_system": true},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if resp.Usage.CacheCreationTokens != 1024 {
		t.Errorf("CacheCreationTokens = %d, want 1024", resp.Usage.CacheCreationTokens)
	}
	if resp.Usage.CacheReadTokens != 2048 {
		t.Errorf("CacheReadTokens = %d, want 2048", resp.Usage.CacheReadTokens)
	}
}
```

Add `"io"` to the test file imports.

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./providers/... -run TestAnthropicProvider_Chat_CacheTokens -v
```

Expected: FAIL — request body does not contain `cache_control` because `buildMessageParams` has no caching path yet. (Usage decoding should already work, since Task 8's `messageToChatResponse` reads `CacheCreationInputTokens` / `CacheReadInputTokens`.)

- [ ] **Step 3: Add cache_control to system blocks when opted in**

Inside `buildMessageParams`, replace the existing system-block construction:

```go
	if systemText != "" {
		params.System = []anthropic.TextBlockParam{{Text: systemText}}
	}
```

with:

```go
	if systemText != "" {
		block := anthropic.TextBlockParam{Text: systemText}
		if cacheOpt, _ := req.Options["prompt_cache_system"].(bool); cacheOpt {
			block.CacheControl = anthropic.CacheControlEphemeralParam{Type: "ephemeral"}
		}
		params.System = []anthropic.TextBlockParam{block}
	}
```

**Executor note:** the exact type for `CacheControl` in the pinned SDK may be a union type wrapper (e.g., `anthropic.CacheControlEphemeralParam{}` wrapped in a union). If `go build` complains, run `go doc github.com/anthropics/anthropic-sdk-go TextBlockParam` and adjust. Either a direct struct-literal assignment or `anthropic.NewCacheControlEphemeral()` will work depending on the version.

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./providers/... -run TestAnthropicProvider_Chat_CacheTokens -v
```

Expected: PASS. If the cache_control field shape is wrong, the error from `go build` plus `go doc` tells you the correct form.

- [ ] **Step 5: Run the whole providers suite**

```bash
go test ./providers/... -v
```

- [ ] **Step 6: Commit**

```bash
git add providers/anthropic.go providers/anthropic_test.go
git commit -m "feat(providers): Anthropic prompt caching with cache-token accounting"
```

---

## Task 12: `anthropic.go` error handling + retry semantics

**Files:**
- Modify: `openharness/providers/anthropic.go`
- Modify: `openharness/providers/anthropic_test.go`

- [ ] **Step 1: Write the failing test**

Append to `providers/anthropic_test.go`:

```go
func TestAnthropicProvider_Chat_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"type":"error","error":{"type":"internal_server_error","message":"boom"}}`))
	}))
	defer srv.Close()

	p := newAnthropicProviderWithBaseURL("anthropic", "test-key", "", srv.URL)

	_, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
	})
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
	if !strings.Contains(err.Error(), "anthropic") {
		t.Errorf("error missing 'anthropic' tag: %v", err)
	}
}

func TestAnthropicProvider_ChatStream_StreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		// Start a message_start then immediately write a malformed event.
		writeSSEEvent(w, "message_start",
			`{"type":"message_start","message":{"id":"x","type":"message","role":"assistant","model":"claude-opus-4-6","content":[],"usage":{"input_tokens":1,"output_tokens":0}}}`)
		writeSSEEvent(w, "error", `{"type":"error","error":{"type":"overloaded_error","message":"server busy"}}`)
	}))
	defer srv.Close()

	p := newAnthropicProviderWithBaseURL("anthropic", "test-key", "", srv.URL)
	_, err := p.ChatStream(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
	}, func(StreamChunk) {})
	if err == nil {
		t.Fatal("expected stream error, got nil")
	}
}
```

- [ ] **Step 2: Run both tests**

```bash
go test ./providers/... -run TestAnthropicProvider_Chat_HTTPError -v
go test ./providers/... -run TestAnthropicProvider_ChatStream_StreamError -v
```

Expected: the first should already pass (the SDK returns an error on non-2xx, and our Chat wraps it with "anthropic chat: %w"). The second should fail or pass depending on whether `stream.Err()` surfaces the overloaded_error event. If it fails, read the SDK source / docs to confirm how error events are surfaced on the streaming iterator, then tighten the detection.

- [ ] **Step 3: Tighten streaming error detection (only if needed)**

If Step 2's streaming test fails because the SDK doesn't raise `stream.Err()` on `event: error`, add an explicit handler in `ChatStream`:

```go
		case anthropic.MessageStreamEvent:
			if e.Type == "error" {
				return nil, fmt.Errorf("anthropic stream event error")
			}
```

(Adjust the type assertion to match whichever event type the SDK uses for errors.)

- [ ] **Step 4: Re-run the whole providers suite**

```bash
go test ./providers/... -v
```

Expected: fully green.

- [ ] **Step 5: Commit**

```bash
git add providers/anthropic.go providers/anthropic_test.go
git commit -m "test(providers): Anthropic non-streaming and streaming error propagation"
```

---

## Task 13: Final verification, lint, push

**Files:** none modified; this task verifies the whole branch.

- [ ] **Step 1: Full test sweep**

```bash
go test ./... -v 2>&1 | tail -30
```

Expected: every openharness package (agent, config, cost, edition, harness, harness/lite, providers, sessions) green.

- [ ] **Step 2: Vet + race check**

```bash
go vet ./...
go test ./providers/... -race -v
```

Expected: no vet findings, race test green.

- [ ] **Step 3: Confirm no ghostfin imports remain**

```bash
grep -rn "swiftdiaries/ghostfin" providers/ || echo "clean"
```

Expected: `clean`.

- [ ] **Step 4: Confirm no `sqliteonly` build tags remain**

```bash
grep -rn "sqliteonly" providers/ || echo "clean"
```

Expected: `clean`.

- [ ] **Step 5: File the fast-follow issue for secret-store integration**

```bash
bd create --type=task --priority=3 \
  --title="providers: wire keychain/SecretStore into ResolveAPIKey" \
  --description="Plan 2 dropped the keychain: scheme from openharness/providers/openrouter.go when extracting from ghostfin. Re-introduce secret resolution via the openharness SecretStore interface (Layer 1) once Plan 4 lands it. Until then, vertical apps pass literal or \$ENV_VAR keys."
```

- [ ] **Step 6: Push branch**

```bash
git push -u origin feat/layer-2-plan-2-providers
```

- [ ] **Step 7: Open PR**

```bash
gh pr create --base main --title "feat(layer-2): Plan 2 — providers (OpenAI-compat + native Anthropic)" \
  --body "$(cat <<'EOF'
## Summary
- Move `providers/{types,models,registry,openai_compat,openrouter}.go` from ghostfin verbatim (sqliteonly build tag stripped; keychain dep dropped from `ResolveAPIKey`)
- Add `providers/anthropic.go` — native `github.com/anthropics/anthropic-sdk-go` provider with real SSE streaming, tool-call reassembly, prompt caching, and cache-token accounting
- Closes the latent `Usage.CacheCreationTokens` / `CacheReadTokens` population gap that no previous ghostfin provider filled

Plan: `docs/superpowers/plans/layer-2/2026-04-13-plan-2-providers.md`
Wave 1B of `docs/superpowers/plans/layer-2/2026-04-13-execution-order.md`.

## Test Plan
- [x] `go test ./providers/... -v` — all green (types, models, registry, openai_compat, openrouter, anthropic)
- [x] `go test ./... -v` — full openharness suite green
- [x] `go vet ./...` — clean
- [x] `go test ./providers/... -race` — clean
- [ ] Post-merge: cut `layer-2-wave-1` tag once both Plan 1 (already merged) and Plan 2 are in `main`
EOF
)"
```

---

## Spec ↔ Plan Self-Review

**Spec requirements covered:**

1. ✅ **Move providers verbatim** — Tasks 2–6 (types, models, registry, openai_compat, openrouter).
2. ✅ **Strip `sqliteonly` build tags** — Tasks 2–6 (Step 2 of each).
3. ✅ **Add native Anthropic provider implementing the existing `Provider` interface** — Task 7 (skeleton + Name/DefaultModel), Task 8 (Chat), Task 9 (ChatStream).
4. ✅ **Real SSE streaming translating `content_block_delta` events to `StreamChunk`** — Task 9.
5. ✅ **Tool-call reassembly server-side** — Task 10.
6. ✅ **Prompt caching via `cache_control` markers on system prompts** — Task 11.
7. ✅ **Populate `Usage.CacheCreationTokens` / `CacheReadTokens`** — Task 11 (test asserts it; `messageToChatResponse` in Task 8 reads the fields).
8. ✅ **Default model constant `claude-opus-4-6`** — Task 7 (`DefaultAnthropicModel` constant).
9. ✅ **Testing via `httptest.NewServer` with canned SSE, no API keys in CI** — Tasks 8–12 all use `httptest.NewServer`; the constructor `newAnthropicProviderWithBaseURL` points the SDK at the test server via `option.WithBaseURL`.
10. ✅ **Error handling** — Task 12 covers non-streaming HTTP errors and streaming event errors.
11. ✅ **No behavior change for ghostfin's existing openai_compat path** — Tasks 5 and 6 are pure moves; Task 5 Step 3 only changes a test string for the (harmless) meta-test that looks for a ghostfin-specific import string.

**Open concerns flagged for the executor:**

- **SDK symbol drift.** The plan's code blocks reference field names (`Message.Usage.CacheCreationInputTokens`, `stream.Current().AsAny()`, `ContentBlockDeltaEvent`, `TextDelta`, `CacheControlEphemeralParam`, `ToolUnionParam`, `ToolParam`, `ToolInputSchemaParam`, `option.WithAPIKey`, `option.WithBaseURL`) taken from the context7 snapshot of `/anthropics/anthropic-sdk-go`. If the pinned version has renamed any of these, executor MUST use `go doc` and adjust — **do not guess**. The tests are the source of truth for end-to-end behavior; the SDK just has to deliver the shapes the tests assert.
- **Keychain scheme drop.** Task 6 Step 3 deleted the `keychain:` branch from `ResolveAPIKey`. The fast-follow (`openharness-1cu`) landed in the same PR as `providers/secret.go`: callers now use `ResolveAPIKeyWithStore(ctx, store, key, fallbackEnv)` with a `harness.SecretStore`, and keys of the form `secret:<name>` are resolved via `SecretStore.GetCredentials`. Ghostfin's Plan 7 rewrite wires a concrete keychain-backed `SecretStore` implementation.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/layer-2/2026-04-13-plan-2-providers.md`. Two execution options:

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task (13 tasks), review between tasks via `superpowers:subagent-driven-development`. Internal parallelism is low for this plan (each task builds on the previous file), so subagent overhead is linear, not fan-out.

**2. Inline Execution** — execute tasks sequentially in this session using `superpowers:executing-plans`, batching Tasks 2–6 (all verbatim moves) as a single checkpoint and Tasks 7–12 (Anthropic TDD cycle) as a second checkpoint.
