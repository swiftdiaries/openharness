## Context

OpenHarness is a Go framework being extracted from `ghostfin/desktop/` over six layers (see `docs/superpowers/specs/2026-04-10-openharness-extraction-design.md`). Layer 1 shipped as `v0.1.0` with eight harness interfaces and Lite implementations. Layer 2 is mid-flight: Plans 1–3 landed (interfaces + concretes for `cost`/`sessions`/`edition`/`config`, providers including a native Anthropic with streaming + prompt caching, tools core + 9 built-ins with `Effects`-based classification + tool-review hardening). Plans 4–8 remain open in their execution plans (MCP+UIBridge, agent loop, `openharness/app/` registration surface, ghostfin import rewrite, examples + `/create-openharness-app` scaffolder).

Today, the only way to assemble an openharness app is to instantiate each Lite store by hand in `main.go`. ghostfin does this; any future vertical would clone-and-adapt. Plan 6 adds an `app.NewApp(AppConfig)` constructor and a registration surface (`Tools()`, `Agents()`, `Telemetry()`, `Frontend()`, `Run()`), but it does not bind the eight stores — that's still the caller's job. The framework's "machine that builds machines" framing is true on paper (the eight interfaces are swappable) but missing the load-bearing primitives that turn it into a real platform: a composition entrypoint, named profiles, and a declarative config surface.

This change adds those three primitives and reframes the remaining extraction work (Layers 3–6) as profile bindings rather than orphan code.

**Stakeholders**: openharness maintainers (this change shapes the public composition API for the rest of the `0.x` series), ghostfin/desktop (consumes the API once Plan 7 lands), future verticals (the scaffolder generates `compose.New(...)` callsites), ghostfin-enterprise (will consume `profiles.Enterprise()` once Layer 5 lands).

**Constraints**:
- Cannot change the eight harness interfaces' method signatures; those evolve under their own changes, including the Layer 1.5 `EventStream.Send` work.
- Cannot break Plan 6's `app.NewApp` once it lands — `compose.New` must produce the same `*app.App` value type.
- API surface frozen for `0.x` with versioned breaks (per the existing extraction-spec policy).
- Must work without Layer 5 enterprise impls existing (Enterprise profile ships as a stub).
- No new external runtime deps beyond what Plans 1–3 already pulled in (`anthropic-sdk-go`, `mcp-go`, `otel`); YAML loader uses `gopkg.in/yaml.v3` which Plan 1 already requires for config.

## Goals / Non-Goals

**Goals:**
- Make `compose.New(profile, opts...)` the documented public path to assembling an `*app.App`. Hand-wired construction is unsupported (kept compiling for `0.x` migration but not advertised).
- Ship `profiles.Lite()` as a re-export of the existing `harness/lite/*` impls under named bindings. No behavior change vs. manual wiring; the profile is wiring code, not new logic.
- Reserve `profiles.Enterprise()` as a stub that errors with a tracking link, so the symbol exists and call sites stabilize before Layer 5 lands.
- Allow custom profiles via `profiles.Register(name, fn)` so verticals/operators can define presets like `my-lite-with-s3`.
- Support a declarative `openharness.yaml` (and JSON variant) that produces the same `*app.App` as the programmatic path.
- Reframe the remaining extraction layers (3, 4, 5, 6) as profile bindings, recorded in `docs/architecture.md` and the upstream extraction spec.

**Non-Goals:**
- Cross-language *runners* (LangGraph/AutoGen/Temporal as native `HarnessRunner` impls). Out of scope per the extraction-spec Scope & Language Boundaries decision.
- Hot-reload of profile bindings at runtime. Profiles bind once at `compose.New` time.
- Dynamic plugin loading (`buildmode=plugin`, `.so` files, RPC plugins).
- Config-format alternates beyond YAML/JSON (no TOML/HCL/CUE).
- A GUI for composition.
- Migrating ghostfin's `desktop/main.go` to `compose.New(...)` — that lands in the Layer 2 Plan 7 / Layer 4 transition, owned by the ghostfin repo.
- Introducing new harness interfaces or modifying existing ones. The eight stay as they are.
- A `provider profile` axis (LLM provider as a profile dimension). Providers stay where Plan 2 put them; profiles are about store/runner bindings.

## Decisions

### D1: `compose.New` returns `*app.App`, not a separate `Composition` type

**Choice:** `compose.New(profile, opts...)` returns the same `*app.App` that Plan 6 introduces. `compose` is a sibling package to `app` that constructs and pre-populates the `*app.App` before the caller registers tools/agents/etc.

**Alternatives considered:**
- A separate `composition.Composition` type that holds the bindings, with a `.Build() *app.App` method. Rejected: extra type, extra ceremony, and the binding-vs-app split is invisible at the call site (callers always want the `*app.App`).
- Putting `compose.New` inside the `app` package as `app.New(profile, opts...)`. Rejected: would force `app` to import `profiles`, which would force `profiles` to import `app` for the `App` type — circular.

**Why it works:** Plan 6's `app.NewApp(AppConfig)` becomes the low-level constructor. `compose.New` is the high-level constructor that calls `app.NewApp` internally then populates store fields from the profile-resolved bindings. Two construction paths, one type, one registration surface afterward.

### D2: Profile is a function type, not an interface

**Choice:** `type Profile func(ctx context.Context, cfg ProfileConfig) (*Bindings, error)`.

**Alternatives considered:**
- An interface `type Profile interface { Bind(ctx, cfg) (*Bindings, error); Name() string }`. Rejected: registration already provides the name (`profiles.Register("name", fn)`); the interface adds boilerplate without adding capability.
- A struct `type Profile struct { Name string; Bind func(...) ... }`. Rejected: same boilerplate problem; also forces verticals to instantiate a struct for every preset.

**Why it works:** Function types compose well, are zero-allocation to pass around, and Go developers are used to seeing `http.HandlerFunc`-style ergonomics. Custom profiles are just `func myProfile(ctx, cfg) (*Bindings, error) { ... }` and `profiles.Register("name", myProfile)`. No type ceremony.

### D3: Bindings struct exposes one field per interface, not a map

**Choice:** `type Bindings struct { Runner harness.HarnessRunner; SessionStore harness.SessionStore; SecretStore harness.SecretStore; /* ... */ }`.

**Alternatives considered:**
- A `map[string]any` keyed by interface name. Rejected: loses static typing; profile authors hit runtime errors instead of compile errors when they forget a binding.
- A reflection-driven binding system. Rejected: hides the dependency graph; harder to grep; mismatches Go's static-typing strengths.

**Why it works:** The eight interfaces are stable and named in the upstream design spec. A flat struct with eight fields makes "I forgot to bind ChannelRouter" a compile error or a clear validation failure (`compose.New` checks every field is non-nil before returning). Profile authors get IDE completion for available bindings.

### D4: YAML loader uses driver-name + options pattern, not direct factory injection

**Choice:** `openharness.yaml` overrides interface bindings by named driver: `secret_store: { driver: vault, options: {...} }`. The loader resolves driver names through a per-interface registry (`drivers.RegisterSecretStore("vault", factory)`).

**Alternatives considered:**
- Inlining factory expressions in YAML (e.g. CUE-like). Rejected: invents a config DSL.
- Requiring all overrides via Go code. Rejected: the whole point of the YAML path is to let operators reconfigure deployments without recompiling.

**Why it works:** Driver registries are a familiar pattern (database/sql, image codecs); they keep the config schema simple (string keys), let factories live in their own packages with their own deps, and let custom profiles register custom drivers without modifying core. Each interface has its own driver registry. Layer 5 enterprise impls register their drivers (`vault`, `s3`, `pg`) and the YAML can mix-and-match them under either profile.

### D5: Enterprise profile ships as an erroring stub

**Choice:** `profiles.Enterprise()` exists in `v0.x` (when this change lands) but its binding function returns an error pointing to the Layer 5 tracking issue. Once Layer 5 lands, the stub is swapped for a real implementation in the same package.

**Alternatives considered:**
- Defer `profiles.Enterprise()` until Layer 5 ships. Rejected: ghostfin-enterprise needs to know the symbol name now to plan its migration; deferring leaves the API surface incomplete.
- Ship a panic-on-call stub. Rejected: panics in framework code are bad form; explicit error is more debuggable.

**Why it works:** Reserving the name now lets the docs, the scaffolder, and the SDK reference `profiles.Enterprise` immediately. The error message points to the Layer 5 roadmap entry so anyone hitting it has a forward path. Replacement is non-breaking.

### D6: Profile populates `RunnerRegistry`; profiles do not replace it

**Choice:** `Bindings.Runner` holds the *default* runner for in-process dispatch. The profile also pre-populates an exported `RunnerRegistry` (held on `*app.App`) with whatever named runners the profile knows about (e.g. `"lite"` → in-process; Enterprise profile would add `"k8s"`, `"lambda"`). Verticals add additional runners via `app.RunnerRegistry().Register(...)` after composition.

**Alternatives considered:**
- Treat the runner as just another store-style binding (single `Bindings.Runner`, no registry). Rejected: `AgentDefinition.Runner` is a string ("lambda", "k8s") that needs name-based dispatch — the registry is load-bearing.
- Make `RunnerRegistry` the only runner surface (no `Bindings.Runner`). Rejected: composition needs a default for `app.Run()` to know what to dispatch when no agent specifies a runner.

**Why it works:** Both surfaces have a job. `Bindings.Runner` is the default in-process runner. `RunnerRegistry` is the dispatch table for agent-declared runners. The profile populates both together; verticals extend the registry without re-doing composition.

### D7: Direct `app.NewApp` is kept exported but documented as low-level

**Choice:** Plan 6's `app.NewApp(AppConfig)` stays exported for the duration of `0.x`. Its godoc directs readers to `compose.New` as the preferred entrypoint. Removing it would force a breaking change on Plan 6 the moment this lands; deferring removal to a `1.0` cleanup.

**Alternatives considered:**
- Unexport `app.NewApp` immediately. Rejected: Plan 6 lands first; this change comes after; flipping export status creates a versioning churn.
- Mark `app.NewApp` deprecated with `// Deprecated:`. Rejected: Plan 6 hasn't landed yet — calling something deprecated before it's released is confusing.

**Why it works:** Documentation is a soft signal; users gravitate to the documented path. Hard removal can wait for a `1.0` API freeze.

## Risks / Trade-offs

- **[Risk] Profile bindings drift from manual wiring.** If `profiles.Lite()` and ghostfin's hand-wired `main.go` initialize the same Lite stores with subtly different defaults, behavior diverges. **Mitigation:** the profile re-exports the existing constructors (`lite.NewLiteSecretStore(...)`) with the same arguments; Layer 1 integration tests run against both paths until Plan 7 deletes the manual wiring.
- **[Risk] YAML schema lock-in.** Once `openharness.yaml` is documented, schema changes are breaking. **Mitigation:** version the schema (`version: 1` field required), reject unknown top-level keys at load time so typos fail loudly, write the schema reference doc with the same care as the Go API.
- **[Risk] Driver registry sprawl.** Each of the eight interfaces gets its own driver registry (`drivers.RegisterSessionStore`, `RegisterSecretStore`, ...). **Mitigation:** keep the registries co-located in `openharness/compose/drivers/`; document driver-name conventions (`lite`, `vault`, `s3`, `pg`); cap to one registration per name per interface (re-register returns an error sentinel).
- **[Risk] Custom profiles obscure the canonical Lite/Enterprise surface.** A vertical defining `my-lite-with-s3` could re-implement most of `profiles.Lite()` and drift. **Mitigation:** `profiles.Lite()` exposes a `Bindings`-returning helper (`profiles.LiteBindings(ctx, cfg)`) that custom profiles can call and then patch — sanctioned composition over copy-paste.
- **[Risk] Stub Enterprise profile masks Layer 5 progress.** If `profiles.Enterprise()` errors silently are swallowed, callers may not notice it never works. **Mitigation:** the stub's error message points to the Layer 5 roadmap entry; the scaffolder rejects `profile: enterprise` until Layer 5 lands (CI gate).
- **[Trade-off] Two construction paths during `0.x`.** `compose.New` (preferred) and `app.NewApp` (low-level) coexist. Slightly more API surface; better migration story. Worth it.
- **[Trade-off] One driver registry per interface.** More boilerplate than a single registry keyed by interface, but compile-time-safe and easier to grep. Worth it.

## Migration Plan

Sequenced PRs, each individually reviewable and shippable:

1. **`compose` package skeleton** — `Option` type, `Bindings` struct, `compose.New` shell that returns an empty `*app.App` (depends on Plan 6 having landed `app.NewApp`). Tests: option deduping, error surfacing.
2. **`profiles` package + `profiles.Lite()`** — `Profile` function type, `ProfileConfig`, `Bindings`, the registry. `profiles.Lite()` re-exports the eight Lite constructors. Tests: bindings non-nil, behavior parity with Layer 1 lite tests.
3. **`profiles.Enterprise()` stub** — function exists, errors with tracking ID. Test: error contains expected substring.
4. **YAML loader + driver registries** — `compose.LoadFromFile`, per-interface driver registries, schema validation, unknown-key rejection. Tests: round-trip YAML ↔ programmatic, schema-violation cases.
5. **Reference vertical update** — once Plan 8's `examples/minimal-vertical/` lands, switch its `main.go` to `compose.New(profiles.Lite(), ...)` and add an `openharness.yaml` next to it. CI gate: scaffolder-generated app uses `compose.New`.
6. **Docs reframe** — `docs/architecture.md`, the extraction spec's "Composition" subsection, `docs/roadmap.md` re-keying.
7. **Layer 4 + Layer 5 dependency** — once Layer 5 lands, replace the Enterprise stub with the real binding (separate change, not part of this one).

**Rollback strategy:** `compose` and `profiles` are net-new packages. Reverting the change drops them and leaves Plan 6's `app.NewApp` path intact. Anyone who adopted `compose.New` migrates back to `app.NewApp` (small mechanical change). No data migration, no schema migration, no on-the-wire change.

**Coordination with Plan 6:** Plan 6 must land before this change starts because composition depends on the `openharness/app` registration surface and `*app.App` existing. This dependency is recorded in `tasks.md`.

**Coordination with Plan 8:** Once both this change and Plan 8 have landed, the scaffolder template flips from `app.NewApp(...)` to `compose.New(profiles.Lite(), ...)`. That flip lands as a follow-up commit on Plan 8's branch or as a small PR after.

**Coordination with ghostfin (Plan 7):** Plan 7's import-rewrite already moves ghostfin's main wiring into the openharness world. Once this change lands, Plan 7 can either (a) ship as-is using `app.NewApp` and migrate to `compose.New` later, or (b) wait and use `compose.New` directly. Decision deferred to Plan 7's executor; either path is reversible.

## Open Questions

- **Should `Bindings` carry a `Telemetry` field?** Telemetry is being interface-inverted in Plan 1 (Layer 2) but its concrete two-layer shape (metadata + spans) lands in Layer 3. If `Bindings` includes telemetry now, Layer 3's refactor is forced through this composition surface. **Tentative answer:** add `Bindings.TraceRecorder` and `Bindings.SpanSink` once Layer 3 designs them; out of scope for this change.
- **Should driver registries be global or per-`compose.New` invocation?** Globals are simpler but interfere with parallel tests. **Tentative answer:** globals with a `compose.NewIsolated(...)` test-only constructor that takes its own driver maps. Decide during PR-2 implementation.
- **Profile composition / inheritance.** Should `profiles.Custom("my-lite-s3", parent: profiles.Lite, override: {...})` exist? Could be useful but inflates surface area. **Tentative answer:** defer; verticals can call `profiles.LiteBindings(...)` then patch fields manually for now.
- **Versioned `openharness.yaml`.** `version: 1` required, but what's the policy for `version: 2`? **Tentative answer:** breaking schema changes bump version; both versions accepted for one minor release; document under `docs/openharness-yaml-versioning.md` once the schema sees its first break.
