# providers

First-depth walkthrough of `providers/`.

## Package summary

`providers` is the LLM provider abstraction: a small `Provider` interface, three concrete impls (Anthropic SDK, OpenAI-compatible HTTP, OpenRouter as a thin wrapper over OpenAI-compat), a registry, secret resolution, and a hardcoded model context-window table. It is the LLM seam for the agent loop.

**Importers (today):** `agent/` consumes its types implicitly (re-defined in `agent.LLMCallInfo` for decoupling); the package itself imports `harness.SecretStore` to support `secret:` URI resolution.

## Files at depth 1

- `providers/types.go` (114 lines) — `Provider` interface, `ChatRequest`/`ChatResponse`/`StreamChunk`, tool/message types, `ParseModelString`.
- `providers/registry.go` (66 lines) — `Registry` keyed by provider name; closes old impls if `io.Closer`.
- `providers/secret.go` (52 lines) — `ResolveAPIKeyWithStore(ctx, harness.SecretStore, key, fallbackEnv)`.
- `providers/anthropic.go` (242 lines) — `AnthropicProvider` using `anthropic-sdk-go`; full streaming, tool use, prompt caching.
- `providers/openai_compat.go` (180 lines) — `OpenAICompatProvider` for any OpenAI-shaped HTTP API; retry on 429/502/503; **no streaming**.
- `providers/openrouter.go` (64 lines) — Legacy `CallOpenRouter()` helper + `ResolveAPIKey(env-or-literal)`.
- `providers/models.go` (24 lines) — Hardcoded `ModelContextWindows` map.
- `providers/anthropic_test.go` (251 lines).
- `providers/openai_compat_test.go` (226 lines).
- `providers/openrouter_test.go` (75 lines).
- `providers/registry_test.go` (63 lines).
- `providers/secret_test.go` (117 lines).
- `providers/anthropic_sse_helper_test.go` (49 lines) — SSE stream construction helper used by other tests.

## Public surface

### Provider abstraction

- `Provider` interface — [`types.go:9-14`](../../providers/types.go) — `Chat`, `ChatStream`, `DefaultModel`, `Name`.
- `ChatRequest` — [`types.go:17-22`](../../providers/types.go) — Messages, Tools, Model, Options map (e.g. `prompt_cache_system`).
- `ChatResponse` — [`types.go:25-31`](../../providers/types.go).
- `StreamChunk` — [`types.go:34-37`](../../providers/types.go) — Content + Done.
- `Message`, `ToolCall`, `Usage`, `ToolDefinition` — [`types.go:39-87`](../../providers/types.go).
- `ParseModelString(providerName, raw)` — [`types.go:93-114`](../../providers/types.go) — Strips routing prefixes (e.g. `openrouter/anthropic/...`).
- `NewTextMessage(role, text)` — [`types.go:61-64`](../../providers/types.go).

### Concrete providers

- **Anthropic** — [`anthropic.go`](../../providers/anthropic.go). `NewAnthropicProvider(apiKey, defaultModel)` ([`anthropic.go:27`](../../providers/anthropic.go)). Full `Provider` impl including real SSE streaming via SDK ([`anthropic.go:81-111`](../../providers/anthropic.go)) and prompt caching via `prompt_cache_system` option ([`anthropic.go:166-191`](../../providers/anthropic.go)).
- **OpenAI-compatible** — [`openai_compat.go`](../../providers/openai_compat.go). `NewOpenAICompatProvider(name, apiBase, apiKey, defaultModel)` ([`openai_compat.go:30`](../../providers/openai_compat.go)). 120s HTTP timeout. Retry loop for 429/502/503 ([`openai_compat.go:45-126`](../../providers/openai_compat.go)). **`ChatStream` falls back to `Chat`** ([`openai_compat.go:129-131`](../../providers/openai_compat.go)).
- **OpenRouter** — `CallOpenRouter` helper ([`openrouter.go:50-63`](../../providers/openrouter.go)) constructs an `OpenAICompatProvider` against `https://openrouter.ai/api/v1`. Not a separate type; legacy convenience surface.

### Registry

- `Registry` — [`registry.go:11-14`](../../providers/registry.go) — `map[string]Provider` + `RWMutex`.
- `NewRegistry()` — [`registry.go:17`](../../providers/registry.go).
- `Register(p Provider)` — [`registry.go:22`](../../providers/registry.go) — Replaces by `Name()`; closes old if `io.Closer`.
- `Get(name)`, `List()`, `Close()` — [`registry.go:34-65`](../../providers/registry.go).

### Secrets

- `SecretScheme = "secret:"` — [`secret.go:21`](../../providers/secret.go).
- `ResolveAPIKeyWithStore(ctx, store harness.SecretStore, key, fallbackEnv)` — [`secret.go:30-52`](../../providers/secret.go) — Three modes: literal, `$ENV`, `secret:name` (looks up `api_key` field, then `name` field).
- `ResolveAPIKey(key, fallbackEnv)` — [`openrouter.go:19`](../../providers/openrouter.go) — env / literal only.

### Models

- `ModelContextWindows map[string]int` — [`models.go:4-16`](../../providers/models.go) — Anthropic 200K, OpenAI 128K, Gemini Pro 1M, etc. Note typo: `claude-haiku-4.5` (dot) and `claude-haiku-4-5` (dash) both present.
- `ContextWindowSize(model) int` — [`models.go:19`](../../providers/models.go) — Default 128000 if unknown.

## Pluggability hooks

To add a provider:

1. Implement `Provider` ([`types.go:9-14`](../../providers/types.go)) — four methods.
2. Optionally implement `io.Closer` if the impl owns connections.
3. `registry.Register(myProvider)`.

The registry pattern means runtime swap-in is straightforward. The `ChatRequest.Options map[string]any` ([`types.go:17`](../../providers/types.go)) is the per-call escape hatch (e.g. `prompt_cache_system`).

## SSE / streaming

- **Anthropic streaming is real** — [`anthropic.go:81-111`](../../providers/anthropic.go) — uses SDK `Messages.NewStreaming(...)` and `.Accumulate()`, forwards text deltas through the `onChunk` callback, emits `{Done: true}` when the stream closes. Test coverage at [`anthropic_test.go:66`](../../providers/anthropic_test.go).
- **OpenAI-compat streaming is not implemented** — [`openai_compat.go:129-131`](../../providers/openai_compat.go) silently falls back to `Chat`. Significant gap if the agent loop expects streaming for OpenRouter/OpenAI.
- `anthropic_sse_helper_test.go` provides `writeSSEEvent` and `writeAnthropicTextStream` helpers for fixture-based streaming tests.

## External deps used

- `github.com/anthropics/anthropic-sdk-go` (+ `option`) — used heavily inside `anthropic.go`. SDK types are not exposed through the `Provider` interface.
- OpenTelemetry: `go.opentelemetry.io/otel/{attribute,codes,trace}` — used by `openai_compat.go` for span attributes (HTTP status, retry count).
- `github.com/swiftdiaries/openharness/harness` — for `SecretStore` interface (used in [`secret.go:8`](../../providers/secret.go)).
- stdlib: `net/http`, `encoding/json`, `io`, `bytes`, `log/slog`, `time`, etc.

## Notable tests

- `TestAnthropicProvider_Chat_Text` — [`anthropic_test.go:23`](../../providers/anthropic_test.go).
- `TestAnthropicProvider_ChatStream_Text` — [`anthropic_test.go:66`](../../providers/anthropic_test.go) — Streaming chunk forwarding.
- `TestAnthropicProvider_Chat_ToolCall` — [`anthropic_test.go:105`](../../providers/anthropic_test.go).
- `TestAnthropicProvider_Chat_CacheTokens` — [`anthropic_test.go:166`](../../providers/anthropic_test.go) — Prompt-caching round trip.
- `TestAnthropicProvider_ChatStream_StreamError` — [`anthropic_test.go:234`](../../providers/anthropic_test.go).
- `TestOpenAICompatProvider_Chat` / `_StripsProviderPrefix` / `_Retry429` — [`openai_compat_test.go`](../../providers/openai_compat_test.go).
- `TestParseModelString` — [`openai_compat_test.go:132`](../../providers/openai_compat_test.go).
- `TestRegistry_RegisterAndGet` / `TestRegistry_ReplaceProvider` — [`registry_test.go`](../../providers/registry_test.go) — Replace-and-close behavior.
- `TestResolveAPIKeyWithStore_*` — [`secret_test.go`](../../providers/secret_test.go) — `secret:` URI dispatch through `harness.SecretStore`.
- `TestChat_NoProviderSpan` — [`openai_compat_test.go:13`](../../providers/openai_compat_test.go) — Pins that the provider does NOT create a span; relies on parent context.

## Open questions / gaps

1. **OpenAI-compat streaming missing.** Agent loops with token-by-token UX cannot use OpenRouter/OpenAI yet.
2. **Anthropic SDK leakage is internal-only.** The `Provider` interface is clean, but `AnthropicProvider` constructor must know SDK types. Acceptable but worth stating.
3. **`models.go` is hardcoded.** New models require a code change. The `claude-haiku-4.5` (dot) entry suggests organic accretion; needs cleanup.
4. **Substring-based fallback context window** isn't done; unknown models silently get 128K.
5. **Asymmetric model-string normalization.** `ParseModelString` exists but `CallOpenRouter` strips `openrouter/` manually ([`openrouter.go:62`](../../providers/openrouter.go)). Consolidate.
6. **No examples of non-Lite secret store**, e.g., Vault. The `secret:` scheme works, but only `lite.LiteSecretStore` exists at depth 1's reach.
7. **Streaming tool-use multi-chunk** not explicitly tested for Anthropic; relies on SDK `.Accumulate`.
8. **`ResolveAPIKey` (no store) is duplicated naming** with `ResolveAPIKeyWithStore`; the former lives in `openrouter.go` for historical reasons. Could be merged.
