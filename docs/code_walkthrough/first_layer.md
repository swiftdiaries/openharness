# openharness — first-layer code walkthrough

A depth-1 review of the openharness Go module, produced by reading every file at the top of each directory and summarizing them. The goal is to let a reviewer judge the framing claim:

> openharness is a pluggable runtime for developing and orchestrating a harness and the agentic AI platform.

Per-package details are split into sibling files; this doc gives the overview, the evidence, and the gaps.

## Purpose & one-paragraph summary

openharness is a Go module that ships **the boundary** of an agentic-AI platform: eight runtime interfaces for sessions, agents, tools, secrets, artifacts, runners, skills, and channels (`harness/`); an LLM provider abstraction with a registry and Anthropic / OpenAI-compat / OpenRouter implementations (`providers/`); a tool framework with effects taxonomy, a registry, a composite multiplexer, and four security primitives (`tools/`); and seam-only contracts for the agent loop's telemetry/knowledge/memory hooks (`agent/`). What it does **not** ship at depth 1 is the agent loop itself, a wiring/assembly layer, or any non-Lite backend. The module is consistent with its own `docs/architecture.md` framing of a layered, registration-driven framework — but several vertical-specific defaults (`~/.ghostfin/`, `GHOSTFIN.md`, ghostfin agent personas) have leaked into `config/`, and a few packages (`edition/`, `cost/`, `sessions/`) have no callers in the rest of the repo.

## Repo map

```
openharness/
├── agent/         Seam-only interfaces for trace / knowledge / memory / prompt context — no loop yet
├── config/        AppConfig + JSON load/save + GHOSTFIN.md discovery
├── cost/          Pricing catalog with override hook + CSV-backed budget tracker
├── docs/          Architecture, roadmap, contributing, this walkthrough, design specs
├── edition/       Lite/Enterprise capability flags backed by atomic.Pointer; unused today
├── harness/       ★ The boundary: 8 interfaces + RunnerRegistry + StoreBackedEventStream
│   └── lite/      Concrete in-process impls (depth-2; not walked here)
├── openspec/      OpenSpec workspace; one active change: platform-builder-runtime
├── providers/     LLM Provider interface + Anthropic / OpenAI-compat / OpenRouter + Registry
├── sessions/      Standalone JSONL session log; does NOT implement harness.SessionStore
├── tools/         ★ Tool interface, effects-aware Registry, Composite, SSRF/shell-deny/scrub
│   └── core/      Built-in tools (depth-2; not walked here)
├── AGENTS.md      bd workflow, non-interactive shell guidance
├── CLAUDE.md      Project rules for AI agents (bd over TodoWrite, push protocol)
├── README.md      One-page overview; alpha; v0.1.0 ships the harness boundary
├── go.mod         module github.com/swiftdiaries/openharness — Go 1.25.0
└── go.sum
```

Per-package walkthroughs (each has its own file in this directory):

- [agent](./agent.md)
- [config](./config.md)
- [cost](./cost.md)
- [docs](./docs.md)
- [edition](./edition.md)
- [harness](./harness.md)
- [openspec](./openspec.md)
- [providers](./providers.md)
- [sessions](./sessions.md)
- [tools](./tools.md)

★ = central to the "pluggable runtime" claim.

## Module identity

- Module: `github.com/swiftdiaries/openharness` ([`go.mod:1`](../../go.mod)).
- Go: `1.25.0` ([`go.mod:3`](../../go.mod)).
- Direct dependencies grouped by purpose ([`go.mod:5-11`](../../go.mod)):
  - **LLM SDK:** `github.com/anthropics/anthropic-sdk-go v1.35.1`.
  - **Tracing:** `go.opentelemetry.io/otel v1.43.0`, `go.opentelemetry.io/otel/trace v1.43.0`.
  - **Networking utilities:** `golang.org/x/net v0.41.0`.
  - **IDs:** `github.com/google/uuid v1.6.0`.
- Indirect deps are JSON tooling pulled in by the Anthropic SDK (`tidwall/{gjson,sjson,match,pretty}`, `cespare/xxhash`).

The dependency footprint is small — no DB drivers, no HTTP framework, no MCP client at depth 1. Anything bigger ships in higher layers.

## Architectural shape at a glance

There are three concentric rings at depth 1:

1. **Boundary** — `harness/`. Eight interfaces plus a runner registry and a polling event-stream adapter. This is where "pluggable" is meant to live.
2. **Building blocks** — `providers/` (LLM seam) and `tools/` (capability seam). Each defines a small interface, a registry, and at least one real implementation.
3. **Seams + scaffolding** — `agent/` (loop interfaces only), `config/` (declarative app config), `cost/` (pricing + spend), `sessions/` (alternate, parallel session log), `edition/` (capability flags). These are mostly defined-but-unwired.

`harness/lite/` is the in-process Lite edition referenced in `docs/architecture.md`. A depth-2 read confirms it implements **seven of the eight** `harness` interfaces — `SessionStore` has no Lite impl in this module. See §Take for the per-surface caveats.

The wiring point — where a gateway constructs all eight subsystems and a `Provider` and a `tools.Registry` and dispatches a run — is **not present at depth 1**. That assembly is the gap between "framework primitives" and "framework you can run."

## Pluggability evidence

Concrete file:line anchors that support the pluggability framing:

- `harness.HarnessRunner` interface — [`harness/interfaces.go:12-18`](../../harness/interfaces.go).
- `harness.RunnerRegistry` named-runner registry — [`harness/runner_registry.go:11-40`](../../harness/runner_registry.go).
- `harness.SessionStore` interface — [`harness/interfaces.go:21-28`](../../harness/interfaces.go).
- `harness.StoreBackedEventStream` polling adapter for remote runners — [`harness/stream_store.go:15-39`](../../harness/stream_store.go).
- `harness.ToolRegistry` global + per-tenant overlay — [`harness/interfaces.go:46-54`](../../harness/interfaces.go).
- `harness.AgentDefinition.Runner` / `FallbackRunner` for per-agent backend selection — [`harness/types.go:79-80`](../../harness/types.go).
- `harness.SandboxPolicy` constraint hook — [`harness/types.go:250-255`](../../harness/types.go).
- `providers.Provider` interface — [`providers/types.go:9-14`](../../providers/types.go).
- `providers.Registry` with replace-and-close-old semantics — [`providers/registry.go:11-65`](../../providers/registry.go).
- `providers.ResolveAPIKeyWithStore` consuming `harness.SecretStore` — [`providers/secret.go:30-52`](../../providers/secret.go) (a load-bearing cross-package coupling).
- `tools.Tool` interface — [`tools/tool.go:62-65`](../../tools/tool.go).
- `tools.Registry` with effects validation — [`tools/registry.go:41-57`](../../tools/registry.go).
- `cost.SetPricingOverrides` — [`cost/pricing.go:67`](../../cost/pricing.go).
- `agent.{TraceRecorder,KnowledgeStore,MemoryStore,PromptContextProvider}` interfaces with no-op defaults — [`agent/interfaces.go`](../../agent/interfaces.go), [`agent/defaults.go`](../../agent/defaults.go).
- Compile-time interface assertions across packages — [`harness/interfaces_test.go`](../../harness/interfaces_test.go), [`agent/interfaces_test.go:12`](../../agent/interfaces_test.go), and (presumed) `harness/lite/satisfy_test.go`.

The pluggability story is genuine at the contract level. Real "swap K8s for Lambda" requires a second `HarnessRunner` impl, which is not in the repo today; the *seam for it* clearly is.

## Agentic-platform evidence

What's present that supports the "agentic AI platform" framing:

- **Tool system with effects taxonomy** — `Read / Mutate / Neutral / Interactive` ([`tools/tool.go:22-44`](../../tools/tool.go)) plus a registry that O(1)-classifies every tool ([`tools/registry.go:67`](../../tools/registry.go)). This is the substrate for an agent loop that gates write actions or modes.
- **Defense in depth for tool execution** — SSRF guard ([`tools/ssrf.go`](../../tools/ssrf.go)), shell deny ([`tools/shell_deny.go`](../../tools/shell_deny.go)), credential scrub ([`tools/scrub.go`](../../tools/scrub.go)), external-content marking ([`tools/external_content.go`](../../tools/external_content.go)). Unusually concentrated for an early-stage repo and a distinguishing feature.
- **Streaming + tool use + prompt caching for Anthropic** — full `ChatStream` with delta callback and cache-control markers ([`providers/anthropic.go:81-191`](../../providers/anthropic.go)).
- **Event types for an agent loop** — `chunk`, `tool.call`, `tool.result`, `ask.user`, `mode.switch`, `checkpoint` constants ([`harness/types.go:9-20`](../../harness/types.go)) and a typed `Event` log model.
- **Multi-tenant agent definitions with versioning + persona files + per-agent runner** — `AgentDefinition` ([`harness/types.go:66-97`](../../harness/types.go)).
- **Skill bundle model** — `Skill`, `SkillMeta`, `SkillRef` ([`harness/types.go:146-170`](../../harness/types.go)) with a `SkillStore` for upload/distribution.
- **Channel router** — webhook in/out for external triggers ([`harness/interfaces.go:57-63`](../../harness/interfaces.go)).

What's **missing** at depth 1:

- **The agent loop itself.** There is no think-act-observe code anywhere in `agent/`, `harness/`, or `providers/`. This is acknowledged by `agent/types.go:1-6` ("the agent loop itself lands in Plan 5") and `docs/roadmap.md` (Layer 2 in progress).
- **A wiring / `App` constructor.** README itself flags this: "The planned `openharness.NewApp(...)` one-call entry point is on the roadmap and isn't shipped." Today consumers must wire every subsystem manually.
- **MCP client / outbound bridge.** Mentioned in roadmap for v0.2.0; not present.

## Cross-cutting observations

1. **Vertical-specific defaults leak beyond `config/`.** `config/` has `~/.ghostfin/workspace`, `GHOSTFIN.md` discovery, and hardcoded `scribe`/`fund-analyst` agents (see [config](./config.md)). The same leak hits `harness/lite/agent.go:32-83` — `NewLiteAgentStoreWithSeeds` ships `fund-analyst`, `analyst`, `scribe`, and `executive-assistant` as Lite seeds. Layer 2 design key decision 8 explicitly commits to zero presets in `openharness/agent` and moves ghostfin agents back to `ghostfin/desktop/agents/` in PR-L2-9, so this is acknowledged debt with a planned fix; worth flagging because today's `harness/lite/` carries it.
2. **Two parallel session implementations.** `sessions/` is JSONL-only, no context, no tenancy; `harness.SessionStore` is the multi-tenant event-log interface. They do not relate to each other. See [sessions](./sessions.md). A reviewer should ask whether `sessions/` is legacy and whether `harness/lite/` provides the canonical Lite session store.
3. **`edition/` and `cost/` are orphan packages.** Both are well-tested and well-shaped, but neither is imported by another package at depth 1. They are framework stubs awaiting consumers.
4. **OpenAI-compat streaming is silently a fallback.** `OpenAICompatProvider.ChatStream` calls `Chat` and synthesizes a single chunk ([`providers/openai_compat.go:129-131`](../../providers/openai_compat.go)). Important for any agent loop that expects token-by-token UX from OpenRouter / OpenAI.
5. **No CI-level whole-module integration test.** Every package's tests are unit tests; there is no end-to-end harness assembly test at depth 1. `harness/lite/satisfy_test.go` only verifies interface conformance — and confirms only seven Lite types satisfy boundary interfaces.
6. **Cross-language is an explicit non-goal.** The extraction spec's §Scope & Language Boundaries rejects Python/LangGraph/Temporal as `HarnessRunner` or `LoopFactory` — non-Go agent frameworks integrate as HTTP+SSE *clients* via Layer 6 only. For a "pluggable runtime" claim, this narrows the meaning: openharness is opinionated toward Claude-Code-shaped interactive agents, not framework-agnostic agent infra.

## Gaps & open questions

Items 1, 2, 3, 5, 6, 7, and 8 from the depth-1 review are resolved by depth-2 reads across `harness/lite/`, `tools/`, `tools/core/`, `providers/`, `cost/`, and `edition/` plus the Layer 2 design spec. Item 4's *runtime* piece (the agent loop reading `ToolEffect` for mode gating) waits on Plan 5 to ship.

1. **Assembly point — answered.** No assembly point ships today. PR-L2-8 introduces `openharness/app` with `NewApp / Tools() / Agents() / Telemetry() / Frontend() / Run()` (registration surface only — full lifecycle lands in Layer 4). PR-L2-11 adds the `/create-openharness-app` scaffolder skill that pins this surface as a committed contract. Today, callers wire each subsystem manually and supply a `LoopFactory` to `LiteRunner`.
2. **`harness/lite/` `SessionStore` — answered.** No `LiteSessionStore` exists; `harness/lite/` ships seven of the eight Layer 1 interfaces. The standalone `sessions/` package is a JSONL log moved as part of L2-0; it does not implement `harness.SessionStore`. The two are unrelated.
3. **`web_fetch` DNS rebind — answered.** Yes, robustly. `tools/core/web_fetch.go:43` uses a custom `DialContext` that calls `tools.ResolveAndCheck(host)` to pin DNS at validation time and dials by resolved IP. The redirect callback at line 67 revalidates with `tools.CheckSSRF`. Pre-fetch validation at line 107 closes the loop.
4. **`ToolEffect` mode gating — partially answered.** `tools.Registry.Register` validates that every definition sets `Effects` ([`tools/registry.go:44-45`](../../tools/registry.go)); every `tools/core/*.go` definition complies. The runtime consumer (loop-detection + `ModePlan` filtering) lives in Plan 5's `agent/tool_loop.go` + `agent/mode.go`, neither shipped. Spec key decision 6 (Layer 2) commits to wiring it.
5. **`cost.Tracker.Record` — answered, still orphan.** Tests are the only callers. Plan 1's `agent.CostTracker` interface points the loop at it; PR-L2-7 (the loop) is where the call site appears.
6. **`edition.Current()` — answered, currently inert.** All callers are in `edition/edition_test.go`. No production code branches on the result.
7. **`MCPServerConfig` consumer — answered.** Read only by `LiteAgentStore.Clone` for deep copy ([`harness/lite/agent.go:255-258`](../../harness/lite/agent.go)). No code attaches the configured MCP servers to a run. That consumer lands in PR-L2-5 (MCP outbound).
8. **`docs/superpowers/specs/` vs `openspec/` — answered.** `docs/superpowers/specs/` is the authoritative design archive (per `docs/contributing.md`); `openspec/` is OpenSpec workspace tooling tracked independently. They coexist by intent.

## Take

The framing claim — "pluggable runtime for an agentic AI platform" — holds at the contract level and is partially backed by working code. After a depth-2 read across `harness/lite/`, `agent/`, `tools/`, `tools/core/`, and `providers/`, plus a sweep of `docs/superpowers/specs/`, the picture is more nuanced than depth 1 alone suggests.

**The boundary is real but Lite covers seven of eight interfaces.** `harness/lite/` ships `LiteAgentStore`, `LiteRunner`, `LiteToolRegistry`, `LiteSecretStore`, `LiteArtifactStore`, `LiteSkillStore`, and `LiteChannelRouter` with compile-time conformance asserted. **There is no `LiteSessionStore`** — `harness.SessionStore` has no consumer in this module. The standalone `sessions/` package does not satisfy the interface. For a "v0.1.0 ships the harness boundary" line, this is the qualifier that matters.

**Within the seven Lite impls, "implemented" is a spectrum.** Several surfaces are deliberately stubbed for single-user desktop use; a reader should not assume parity:

- `LiteRunner.Resume` returns `ErrNotFound` — no durable sessions in embedded mode ([`harness/lite/runner.go:140-142`](../../harness/lite/runner.go)). The runner is goroutine-backed and accepts a caller-supplied `LoopFactory`; the agent loop is bring-your-own.
- `LiteSecretStore.MintScopedToken` returns the static string `"lite-passthrough"` ([`harness/lite/secret.go:67-70`](../../harness/lite/secret.go)) — capability tokens are non-functional in Lite.
- `LiteArtifactStore.Presign` ignores the requested TTL and returns a `file://` URL ([`harness/lite/artifact.go:103-120`](../../harness/lite/artifact.go)) — no real expiry.
- `LiteChannelRouter.HandleInbound` always errors; `RegisterWebhook` returns an empty URL; `SendOutbound` is a no-op unless a `SendFunc` is injected ([`harness/lite/channel.go:48-87`](../../harness/lite/channel.go)).
- `LiteSkillStore.ListPlatform` returns an empty slice ([`harness/lite/skill.go:38-41`](../../harness/lite/skill.go)) — no platform skills are bundled.
- Metadata for agents, tenant tools, channel registrations, and in-memory secret overrides is RAM-only and is lost on restart; only artifacts and skill prompt files persist to disk.

These are reasonable for a desktop tier, but they sharpen what "ships the boundary" means: the *contract* and the *seam* are real; the *capabilities* behind each seam range from honest in-process impls (agents, tools, runner, artifacts) to passthrough stubs (scoped tokens, inbound channels, platform-skill catalog).

**The ghostfin leak is broader than `config/`.** `harness/lite/agent.go:32-83` seeds five agents — `harness`, `fund-analyst`, `analyst`, `scribe`, `executive-assistant` — four of which are ghostfin personas inside framework code. Layer 2 design key decision 8 commits to "zero preset agent definitions in `openharness/agent`" and PR-L2-9 moves these back to ghostfin; this is acknowledged debt, but a reviewer assessing "domain-agnostic framework" today should see it.

**Layer 2 is in flight, not absent.** The depth-1 reviewer who reads only this module sees no `agent/loop.go` and concludes the loop is missing — accurate. But per the extraction spec's progress table, Layer 2 ships across 12 PRs (PR-L2-0 … PR-L2-11) grouped into plans, and Plans 1–3 (`agent/` interfaces, `providers/`, `tools/` with effects + `tools/core/`) are merged. What's still out for v0.2.0: Plan 4 (MCP outbound + UIBridge), Plan 5 (the actual `agent/loop.go`), Plan 8 (`openharness/app` registration surface), Plan 11 (the scaffolder skill), and the **Layer 1.5 break** — `harness.Event` field additions, `EventStream.Send`, and a `LoopFactory` signature change that intentionally breaks v0.1.0. Saying "agent loop missing" without naming Plan 5 understates the structure of the in-flight work.

**Cross-language scope is opinionated.** The depth-1 review missed the spec's §Scope & Language Boundaries: Go-only `HarnessRunner` / `LoopFactory` is an explicit non-goal. Python/LangGraph/AutoGen/Temporal as runners or loops are *rejected*, not deferred; non-Go consumers integrate as HTTP+SSE clients via Layer 6 only. This narrows what "pluggable" means in practice and is a load-bearing design call worth surfacing in any review of the framework framing.

**Two corrections to the depth-1 cross-cutting observations:**

- The "5 layers vs 6 layers" doc-inconsistency claim does not survive verification. `README.md`, `docs/architecture.md`, `docs/roadmap.md`, and the extraction spec all say six. Earlier "5-layer" framing was in stale memory notes, not in any committed doc.
- "OpenAI-compat streaming silently a fallback" is correct *and* explicitly committed: Layer 2 design key decision 3 keeps it stubbed as a fast-follow ticket. It's a known stub, not an oversight.

**Depth-2 verification of Plans 1–3.** The merged plans ship with the rigor the specs commit to:

- **Plan 2 (providers).** `providers/anthropic.go` uses the SDK's real streaming API (`p.client.Messages.NewStreaming` at [`providers/anthropic.go:87`](../../providers/anthropic.go)), reads `Usage.CacheCreationInputTokens` / `Usage.CacheReadInputTokens` into the framework's `Usage` ([`providers/anthropic.go:234-239`](../../providers/anthropic.go)), and sets `cache_control` on the system prompt when `req.Options["prompt_cache_system"]` is true ([`providers/anthropic.go:152-156`](../../providers/anthropic.go)). The Plan 2 cache-token-accounting goal is met. By contrast, `providers/openai_compat.go:128-130` `ChatStream` is a three-line fallback to `Chat` — the streaming callback is never invoked, not even with a single synthesized chunk; the depth-1 framing of "synthesizes a single chunk" is more generous than the code.
- **Plan 3 (tools) is wired with teeth.** `tools.Registry.Register` errors when any definition has `ToolEffectUnspecified` ([`tools/registry.go:44-45`](../../tools/registry.go)); every `ToolDefinition` literal in `tools/core/*.go` populates `Effects` (16 definitions across 9 files). `tools/core/web_fetch.go` is the strongest defense-in-depth piece in the repo: a custom `DialContext` calls `tools.ResolveAndCheck(host)` to pin DNS at validation time and dials by resolved IP ([`tools/core/web_fetch.go:43`](../../tools/core/web_fetch.go)); `CheckRedirect` revalidates ([`tools/core/web_fetch.go:67`](../../tools/core/web_fetch.go)); the tool re-checks before fetching ([`tools/core/web_fetch.go:107`](../../tools/core/web_fetch.go)). Output sanitization is layered across all read-style tools: `SanitizeRead` / `SanitizeExternal` is applied in `filesystem` `read_file`, `memory_search`/`memory_get`, `exec` stdout/stderr, `web_fetch`, `web_search`, and `knowledge_graph_search`. `shell_deny` is enforced in `exec` ([`tools/core/exec.go:98`](../../tools/core/exec.go)).
- **Plan 1 (agent/ interfaces) is half-real.** `agent.MemoryStore` has a concrete impl in `tools/core/memory.go:Memory`. The other three — `TraceRecorder`, `KnowledgeStore`, `PromptContextProvider` — have only the no-op defaults from `agent/defaults.go`; no real impl exists in this module yet.

**The pattern depth-2 surfaces — pluggability seams with no in-module consumers.** Several affordances exist on the boundary but have nothing wiring them within openharness today:

- `harness.RunnerRegistry` has no in-module callers — only the definition and tests.
- `harness.StoreBackedEventStream` has no in-module callers.
- `tools.Composite` is never instantiated outside its own tests.
- `providers.ResolveAPIKeyWithStore` (the depth-1 review's "load-bearing cross-package coupling") is not called by any provider — [`providers/openrouter.go:17`](../../providers/openrouter.go) references it only in a comment.
- The standalone `sessions/` package has zero importers in the module.
- `harness.AgentDefinition.MCPServers` is read in exactly one place — `LiteAgentStore.Clone` for deep copy ([`harness/lite/agent.go:255-258`](../../harness/lite/agent.go)). Nothing reads it to actually attach MCP servers to a run — that consumer lands in PR-L2-5.
- `cost.Tracker.Record` and `edition.Current()` have no production callers (tests only).

The depth-1 review flagged some of this individually (`cost`/`edition` as orphan packages; `sessions/` as parallel/unrelated). The broader shape is worth naming: **today, openharness ships the contracts but very few of them are activated within the module itself.** The customer wiring all of these together is ghostfin's desktop on the legacy path, and (post-Plan 5) `agent/loop.go` plus PR-L2-8's `openharness/app` for the framework path. The eight-interface boundary plus its registries plus its polling adapters *will* be load-bearing for both paths, but they aren't yet.

**Net.** The framework claim is honest at the contract level; depth-2 confirms Plans 2 and 3 are wired with the rigor the specs commit to (real streaming, cache-token accounting, DNS pinning + revalidation, effects validation, output scrubbing). The remaining gap to a usable framework is genuinely Plan 5 (`agent/loop.go`), Plan 8 (`openharness/app` registration surface), and PR-L2-5 (the MCP consumer of `AgentDefinition.MCPServers`). The unused-seams pattern is a yellow flag, not a red one — contracts written ahead of consumers is a defensible Layer 1 / Layer 1.5 strategy when the consumers are tracked in a documented plan chain. A reviewer should weight runtime maturity below what the contracts alone suggest, but should also recognize that the contracts being there at all is what makes the rest of the chain reviewable.
