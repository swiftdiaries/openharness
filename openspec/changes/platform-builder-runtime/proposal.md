## Why

Layer 1 shipped (`v0.1.0`): eight swappable harness interfaces — `HarnessRunner`, `SessionStore`, `SecretStore`, `ArtifactStore`, `ToolRegistry`, `ChannelRouter`, `AgentStore`, `SkillStore` — plus Lite implementations of each. Layer 2 is mid-flight: Plans 1–3 landed (interfaces+concretes, providers including a native Anthropic with streaming and prompt caching, tools core + 9 built-ins with `Effects`-based classification); Plans 4–8 (MCP+UIBridge, agent loop, `openharness/app` registration surface, ghostfin import rewrite, examples+scaffolder) remain open in their execution plans. The interfaces and the Plan 6/8 deliverables prove "pluggable" mechanically — but the framing in docs and code is still **"the framework extracted from ghostfin"**, not **"a platform that builds platforms."**

The forward goal we want to commit to: openharness is **a machine for building machines** — a runtime where any agentic application is composed by snapping together a runner, the seven stores, a tool catalog, and a vertical's domain code. To make that real (not just true-on-paper), three primitives are missing on top of what Layers 1–6 already plan:

1. **A composition entrypoint.** Today every vertical wires the eight stores by hand in `main.go` (ghostfin does it; any future vertical would clone-and-adapt). Plan 6 adds `app.NewApp(AppConfig{...})` which holds the registration surface but does not bind the stores. There is no `compose.New(profile, opts...)` call that returns an app with all eight interfaces bound to a sensible default.
2. **Named profiles as first-class artifacts.** "Lite" and "Enterprise" exist as directory names (`harness/lite/`, future `harness/enterprise/`) but not as loadable presets. There is no `profiles.Lite()` you can pass in. Verticals/operators cannot define their own profile (e.g., "Lite-with-S3-artifacts-and-keychain-secrets") without duplicating wiring code.
3. **Declarative composition.** Today's only composition surface is Go code. A platform-builder runtime should also accept an `openharness.yaml` (profile + per-interface overrides + tool/agent registration) so non-Go operators can drive deployments and so the scaffolder can emit one.

Reframing now (rather than after Layers 3–6 land) means each remaining layer's deliverable lands as a **profile binding**, not as orphan code that needs gluing later. Layer 5's enterprise impls become "the Enterprise profile binds these"; Layer 4's app scaffold becomes "the Lite profile mounts this lifecycle"; Layer 6's SDK becomes "a transport over the composed app." The platform-builder lens is the through-line that connects the remaining work.

## What Changes

- **NEW: `openharness/compose/`** — programmatic composition entrypoint. `compose.New(profile, opts...) (*app.App, error)` returns an `App` with all eight interfaces bound. Verticals override individual bindings via `compose.WithRunner(...)`, `compose.WithSessionStore(...)`, etc. Layered on top of Plan 6's `openharness/app/` package — `compose` produces what `app.NewApp` then exposes.
- **NEW: `openharness/profiles/`** — named, composable presets. Each profile is `func(ctx context.Context, cfg ProfileConfig) (*Bindings, error)` that returns a default binding for every interface. Ships with `profiles.Lite()` (rebinds the existing `harness/lite/*` impls behind the named API) and a stub `profiles.Enterprise()` that errors with a helpful message until Layer 5 lands. Custom profiles register via `profiles.Register("name", fn)`.
- **NEW: declarative composition config.** `openharness.yaml` schema (loader in `compose/config.go`) selects a profile, overrides per-interface bindings by named driver (e.g., `secret_store: vault`), and lists tool/agent registrations. The loader returns the same `*app.App` as the programmatic path. JSON is supported via the same loader for tooling/CI.
- **REFRAME: extraction layers as profile bindings.** The remaining Layer 3–5 deliverables are re-keyed to "what they bind into which profile":
  - Layer 3 telemetry/cost/sessions concretes bind into `profiles.Lite()` (SQLite metadata + DuckDB sink) and `profiles.Enterprise()` (Postgres metadata + OTLP sink).
  - Layer 4 app scaffold (Wails + gateway) becomes a Lite-profile lifecycle option, not a hardcoded path.
  - Layer 5 enterprise impls land as Enterprise-profile bindings.
  - Layer 6 SDK transports compose against an already-built `*app.App` (in-process for Lite, HTTP+SSE for Enterprise) — no reach-around into stores.
- **EXTEND: `harness.RunnerRegistry` profile awareness.** `RunnerRegistry.Register` keeps its current signature; profiles populate it during binding. Document that registry is the runner-pluggability surface; non-Go runners (Lambda, K8s job, future HTTP-driven worker) compose under the same registry. No protocol change for in-process runners.
- **DOC: foreground the platform-builder framing.** `docs/architecture.md` rewrites its lead section around composition. The extraction spec (`docs/superpowers/specs/2026-04-10-openharness-extraction-design.md`) gets a "Composition" section linking to this change. The roadmap re-keys remaining Layer 3–6 work to profile bindings.
- **TIE-IN with Plan 6 (`openharness/app/`).** This change does not duplicate Plan 6: it consumes it. `app.App` continues to expose `Tools()`, `Agents()`, `Telemetry()`, `Frontend()`, `Run()`. `compose.New` is what builds the `*app.App` and pre-populates its store fields before returning. Plan 6 lands `app.NewApp(AppConfig)` for hand-wired use; this change adds `compose.New(profile, opts...)` for profile-driven use, and both end at the same `*app.App` value.
- **TIE-IN with Plan 8 (`/create-openharness-app`).** The scaffolder emits a `main.go` that calls `compose.New(profiles.Lite(), ...)` and an `openharness.yaml` next to it. The reference vertical at `examples/minimal-vertical/` exercises the composition path end-to-end.

**Non-goals** (deferred to follow-up changes):
- Cross-language *runners* (LangGraph/AutoGen/Temporal as native `HarnessRunner` impls). Cross-language *consumers* via Layer 6 HTTP+SSE remain the supported path; the Scope & Language Boundaries decision in the upstream extraction spec stands.
- Hot-reload of profile bindings at runtime, dynamic plugin loading (`buildmode=plugin`, `.so` files), config-format alternates beyond YAML/JSON, and a GUI for composition.
- Changing the eight harness interfaces' method signatures. Composition is over them, not redesign of them. Breaking interface changes remain in their own changes, including the Layer 1.5 `EventStream.Send` work.
- Migration of ghostfin's `desktop/main.go` to `compose.New(...)` — that lands in the Layer 2 Plan 7 / Layer 4 transition, owned by the ghostfin repo.

## Capabilities

### New Capabilities

- `app-composition`: programmatic and declarative assembly of an openharness app. The `compose` package, the `compose.New(profile, opts...)` entrypoint, the option functions (`WithRunner`, `WithSessionStore`, ...), the `openharness.yaml` schema, and the loader that maps both onto the same `*app.App` value. Includes the contract that overrides win over profile defaults.
- `runtime-profiles`: named, composable presets that bind every harness interface to a default implementation. The `profiles.Profile` contract, the `profiles.Lite()` / `profiles.Enterprise()` built-ins, the registry of named profiles, the `ProfileConfig` shape (data dir, edition, default LLM provider), and the rules for verticals/operators defining custom profiles.

### Modified Capabilities

<!-- None. openspec/specs/ has no committed specs yet; all surfaces here are net-new. The Plan 6 registration surface (Tools/Agents/Telemetry/Frontend/Run) and Plan 8 scaffolder + reference vertical are tracked under their existing Layer 2 plans, not as modifications here. -->

## Impact

- **New packages**: `openharness/compose/` (entrypoint + options + config loader), `openharness/profiles/` (Profile contract, Lite + Enterprise built-ins, registry).
- **Touched packages**: `harness/runner_registry.go` gains documentation only — no signature change. `harness/lite/*` stays as the underlying impl that `profiles.Lite()` re-exports under named bindings. `openharness/app/` (landing under Plan 6) gains a constructor path that `compose.New` uses to populate stores.
- **Public API surface (frozen for `0.x` with versioned breaks)**: `compose.New`, `compose.With*` options, `profiles.Profile` interface, `profiles.Lite/Enterprise/Register`, the `openharness.yaml` schema. Breaks tracked per-version per the existing extraction-spec policy.
- **Documentation**:
  - `docs/architecture.md` — lead section rewrites around composition (platform-builder framing).
  - `docs/superpowers/specs/2026-04-10-openharness-extraction-design.md` — adds a "Composition" subsection in §System Architecture; updates §Migration Strategy to key Layers 3–5 to profile bindings.
  - `docs/roadmap.md` — re-keys remaining work to profile bindings.
- **OpenSpec**: this change's `tasks.md` tracks the `compose` package, `profiles` package, YAML loader, documentation reframe, and dependencies on Layer 2 Plans 6 and 8.
- **Consumers**:
  - `ghostfin/desktop/main.go` — when Plan 7 lands the import rewrite, the new `main.go` will use `compose.New(profiles.Lite(), ...)` rather than ad-hoc store wiring. This change provides the API; ghostfin migration is out of scope here.
  - Future verticals + the scaffolder-generated reference vertical — entry path is `compose.New(profiles.Lite(), ...) + app.Tools().Register(...)`.
  - `ghostfin-enterprise` — once Layer 5 lands, wiring becomes `compose.New(profiles.Enterprise(), ...)`.
- **Out of scope (named here so they don't sneak in)**: hot-reload, plugin loading, alternate config formats (TOML/HCL), composition GUI, telemetry-two-layer refactor (Layer 3), Wails app lifecycle (Layer 4), enterprise impls (Layer 5), HTTP+SSE transport (Layer 6).
