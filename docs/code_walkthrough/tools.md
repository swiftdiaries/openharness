# tools

First-depth walkthrough of `tools/`.

## Package summary

`tools` defines the agent tool system: a small `Tool` interface, a `Registry` that classifies tools by **effect** (Read / Mutate / Neutral / Interactive), a `Composite` multiplexer, and four orthogonal security primitives (SSRF guard, shell deny patterns, credential scrubbing, external-content marking). Built-in tools live under `tools/core/` (depth-2; not walked here).

This is the primary extension point for any vertical that wants to give an agent new capabilities.

**Importers (today):** `tools/core/` (built-in implementations); external verticals would import `tools` directly to register custom tools.

## Files at depth 1

- `tools/tool.go` (66 lines) — `ToolEffect` enum, `ToolDefinition`, `Tool` interface.
- `tools/registry.go` (126 lines) — Effects-aware registry; rejects `ToolEffectUnspecified`; rejects name collisions.
- `tools/composite.go` (37 lines) — `Composite` multiplexer.
- `tools/ssrf.go` (116 lines) — URL/IP/DNS guard with private-range list and DNS pinning helper.
- `tools/shell_deny.go` (37 lines) — Regex deny list for destructive shell patterns.
- `tools/scrub.go` (37 lines) — Credential-redaction regex set.
- `tools/external_content.go` (17 lines) — Wraps untrusted text in delimiter + warning.
- `tools/README.md` (96 lines) — Authoring guide; canonical reference for effects + registration.
- `tools/tool_test.go` (42 lines).
- `tools/registry_test.go` (32 lines).
- `tools/composite_test.go` (68 lines).
- `tools/ssrf_test.go` (60 lines).
- `tools/shell_deny_test.go` (33 lines).
- `tools/scrub_test.go` (47 lines).

## Subdirectories at depth 1

`tools/core/` (NOT walked here). Filenames observed:

- `core.go`, `ask_user.go`, `exec.go`, `filesystem.go`, `memory.go`, `knowledge_graph.go`, `knowledge_graph_types.go`, `subagent.go`, `tasks.go`, `web_fetch.go`, `web_search.go`, `external_output.go`.
- Test files for each.

These are the bundled built-in tools, registered together by `core.Register(registry, core.Config{...})`.

## Public surface

### Tool abstraction

- `ToolEffect` enum — [`tool.go:22-44`](../../tools/tool.go) — `ToolEffectUnspecified`, `ToolEffectRead`, `ToolEffectMutate`, `ToolEffectNeutral`, `ToolEffectInteractive`.
- `ToolDefinition` — [`tool.go:50-57`](../../tools/tool.go) — Name, Description, Parameters (`json.RawMessage`), Effects (`json:"-"`).
- `Tool` interface — [`tool.go:62-65`](../../tools/tool.go) — `Definitions() []ToolDefinition`, `Execute(ctx, name, args json.RawMessage) (json.RawMessage, error)`.

### Registry

- `Registry` — [`registry.go:21-34`](../../tools/registry.go) — Three maps: `tools`, `categories`, `effects`.
- `NewRegistry()` — [`registry.go:28`](../../tools/registry.go).
- `Register(category, Tool) error` — [`registry.go:41-57`](../../tools/registry.go) — Validates non-Unspecified effects, atomic rollback on collision.
- `Get(name) Tool` — [`registry.go:60`](../../tools/registry.go).
- `EffectsFor(name) ToolEffect` — [`registry.go:67`](../../tools/registry.go) — O(1).
- `Unregister(name)` — [`registry.go:72`](../../tools/registry.go).
- `Categories() []ToolCategory` — [`registry.go:79`](../../tools/registry.go).
- `ToolsByCategory(cat) []Tool` — [`registry.go:99`](../../tools/registry.go).
- `ToolsByNames(names ...string) []Tool` — [`registry.go:115`](../../tools/registry.go).

### Composite tools

- `Composite` — [`composite.go:9-12`](../../tools/composite.go) — `[]Tool`.
- `NewComposite(tools ...Tool)` — [`composite.go:15`](../../tools/composite.go).
- `(*Composite).Definitions()` merges — [`composite.go:18-24`](../../tools/composite.go).
- `(*Composite).Execute(ctx, name, args)` routes by name — [`composite.go:27-36`](../../tools/composite.go).

### Security primitives

- **SSRF guard** — [`ssrf.go:52-87`](../../tools/ssrf.go) — `CheckSSRF(rawURL) error`. Blocked hostnames at [`ssrf.go:10-26`](../../tools/ssrf.go) (`localhost`, `metadata.google.internal`, `*.local`, `*.internal`, `*.localhost`). Private CIDRs at [`ssrf.go:28-48`](../../tools/ssrf.go) (RFC 1918 + IPv6 link-local + ULA + carrier NAT). DNS pinning helper `ResolveAndCheck(host)` at [`ssrf.go:92-115`](../../tools/ssrf.go).
- **Shell deny** — [`shell_deny.go:8-27`](../../tools/shell_deny.go) — 9 regex patterns: `rm -rf`, `rm -fr`, `rmdir /s`, `mkfs`, `dd if`, `shutdown|reboot|poweroff|halt|init [06]`, fork-bombs `:(){...}` and `.(){...}`, git destructive (`git push --force` / `git reset --hard` / `git clean -f`).
- **Credential scrubbing** — [`scrub.go:5-29`](../../tools/scrub.go) — 14 regex patterns across 9 categories: OpenAI `sk-...`, Anthropic `sk-ant-...`, GitHub PATs (5 prefixes: `ghp_`/`gho_`/`ghu_`/`ghs_`/`ghr_`), AWS `AKIA...`, bearer tokens, generic `api_key=...`/`token=...`/`secret=...`/`password=...`, connection strings (postgres/mysql/mongodb/redis/amqp), env-var-style values (KEY/SECRET/CREDENTIAL/PRIVATE + DSN/DATABASE_URL family), 64+ hex.
- **External content marking** — [`external_content.go:14-16`](../../tools/external_content.go) — `WrapExternalContent(s)` returns delimited untrusted text with explicit security notice for the LLM.

## Pluggability hooks

To register a tool, a vertical:

1. Implements `Tool` ([`tool.go:62-65`](../../tools/tool.go)).
2. Sets `Effects` on every `ToolDefinition` ([`tool.go:50-57`](../../tools/tool.go)) — must not be Unspecified.
3. Calls `registry.Register(category, t)` ([`registry.go:41`](../../tools/registry.go)).

The README [`tools/README.md`](../../tools/README.md) walks through a minimal example and the optional-fields-mean-skip pattern used by `core.Register`.

## Defense-in-depth posture

This is unusually concentrated for an agentic platform:

- SSRF guard sits in front of any HTTP-fetching tool.
- Shell deny is a second layer below an exec allowlist (presumed in `tools/core/exec.go`).
- Output scrubber prevents credential exfiltration via tool results.
- External-content delimiter signals the LLM that fetched text is adversarial input, not instruction.

These are independent, regex/list-driven, and designed for layering.

## External deps used

stdlib only at depth 1: `context`, `encoding/json`, `fmt`, `net`, `net/url`, `regexp`, `sort`, `strings`. No `golang.org/x/net` despite SSRF guard — the package re-implements private-range checks.

## Notable tests

- `TestToolEffectZeroValueIsUnspecified` — [`tool_test.go`](../../tools/tool_test.go) — Defensive default.
- `TestToolDefinitionEffectsNotSerialized` — [`tool_test.go`](../../tools/tool_test.go) — Pins `json:"-"` on `Effects`.
- `TestRegister_RejectsNameCollision` — [`registry_test.go`](../../tools/registry_test.go).
- `TestComposite{Definitions,Execute,ExecuteNotFound}` — [`composite_test.go`](../../tools/composite_test.go).
- `TestCheckSSRF` — [`ssrf_test.go`](../../tools/ssrf_test.go) — 9 cases incl. IPv6 loopback, `.local`, `.internal`, GCP metadata.
- `TestResolveAndCheck_*` — [`ssrf_test.go`](../../tools/ssrf_test.go) — DNS-pinning helper.
- Shell-deny + scrub coverage tables — [`shell_deny_test.go`](../../tools/shell_deny_test.go), [`scrub_test.go`](../../tools/scrub_test.go).

## Open questions / gaps

1. **No registry test for the effect-validation path** — `registry_test.go` covers collisions but not Unspecified rejection.
2. **`Composite.Definitions` does not deduplicate** — relies on registry upstream for collision prevention. If two composed tools both expose name `web_fetch`, the first wins silently.
3. **DNS rebind protection — answered.** `tools/core/web_fetch.go` wires `ResolveAndCheck` into a custom `DialContext` ([line 43](../../tools/core/web_fetch.go)) that pins DNS at validation time and dials by resolved IP. The redirect callback ([line 67](../../tools/core/web_fetch.go)) revalidates with `CheckSSRF`; pre-fetch validation ([line 107](../../tools/core/web_fetch.go)) closes the loop. The seam is real and consumed.
4. **Hex-64 scrub false positives**: SHA-256 hashes, build IDs, and commit hashes get redacted. The test set explicitly accepts this.
5. **Effect → loop semantics**: README references "Plan 5's loop" using effects to gate ModePlan; that loop isn't in the repo at depth 1. The data is in place; the consumer is missing.
6. **Built-ins matrix**: `tools/core/` has a referenced canonical "D4 matrix" test; depth-2 walkthrough should map every built-in to its declared effect.
