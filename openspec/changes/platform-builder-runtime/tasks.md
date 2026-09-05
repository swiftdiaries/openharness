## 1. Pre-flight

- [ ] 1.1 Confirm Layer 2 Plan 6 (`openharness/app/` registration surface) has landed on `main` and `app.NewApp(AppConfig)` returns a usable `*app.App`. Keep this change blocked until that prerequisite is complete.
- [ ] 1.2 Review this task list against `proposal.md`, `design.md`, and both capability specs; add any missing implementation or verification tasks before starting PR-1.
- [ ] 1.3 Add a `replace` directive in any local `go.work` only if the user already uses one; otherwise rely on the published openharness module. (Memory note: this repo prefers real tags over `replace`/`go.work use`.)
- [ ] 1.4 Snapshot the current `go test ./...` baseline on `main` so post-change diffs are interpretable. Record the test count in the PR-1 description.

## 2. PR-1 — `compose` package skeleton

- [ ] 2.1 Create `openharness/compose/compose.go` with the `Option` function type, `Bindings` struct (eight nullable interface fields), and a stub `compose.New(profile Profile, opts ...Option) (*app.App, error)` that returns a non-nil `*app.App` from `app.NewApp(AppConfig{})` without yet invoking the profile.
- [ ] 2.2 Add `compose.WithDataDir`, `compose.WithEdition`, `compose.WithDefaultProvider` configuration options that populate a private `composeConfig` struct consumed by `compose.New` (these flow into `ProfileConfig` in PR-2).
- [ ] 2.3 Add `compose.With{Runner,SessionStore,SecretStore,ArtifactStore,ToolRegistry,ChannelRouter,AgentStore,SkillStore}` override option functions. Each option records the override in the private config; conflicting options for the same interface return an error from `compose.New` naming the duplicated option.
- [ ] 2.4 Add `compose/compose_test.go` with: option deduping (every `With*` covered), error surfacing on conflicting overrides, nil-profile rejection. Use a synthetic `Profile` that returns a fully-bound `*Bindings` for these tests; do not depend on `profiles.Lite()` yet.
- [ ] 2.5 Add `compose/doc.go` with package-level godoc that names `compose.New` as the documented public entrypoint and links to `profiles.Lite`.
- [ ] 2.6 Run `go test ./compose/... -race` and `go vet ./compose/...`. Both must be green. Open the PR; require review before merging.

## 3. PR-2 — `profiles` package + `profiles.Lite()`

- [ ] 3.1 Create `openharness/profiles/profile.go` with the `Profile` function type (`func(ctx context.Context, cfg ProfileConfig) (*Bindings, error)`), `ProfileConfig` struct (`DataDir string`, `Edition edition.Edition`, `DefaultProvider string`, `Options map[string]any`), and the public `Bindings` struct (re-imported into `compose` to avoid circular deps; canonical home is `profiles`).
- [ ] 3.2 Create `openharness/profiles/registry.go` with `Register(name string, p Profile) error`, `Lookup(name string) (Profile, error)`, an unexported `defaultRegistry`, and `ErrProfileExists` / `ErrProfileNotFound` sentinel errors. Re-registering an already-registered name returns `ErrProfileExists` and leaves the existing registration untouched.
- [ ] 3.3 Create `openharness/profiles/lite.go`. Define `Lite() Profile` returning a closure that constructs the eight Lite stores using existing `harness/lite/*` constructors. Pass `cfg.DataDir` to path-using stores (`LiteSessionStore`, `LiteArtifactStore`, `LiteSkillStore`); ignore in stores that don't need it. Run `init()` to call `Register("lite", Lite())`.
- [ ] 3.4 Add `profiles.LiteBindings(ctx, cfg) (*Bindings, error)` exported helper so custom profiles can compose-and-patch (per the design doc's "sanctioned composition over copy-paste" mitigation).
- [ ] 3.5 Add a `profiles.validate(*Bindings) error` helper invoked by `compose.New` after the profile returns. It checks every interface field is non-nil; returns an error naming the missing field if not.
- [ ] 3.6 Wire `compose.New` to actually invoke the profile: build `ProfileConfig` from `composeConfig`, call `profile(ctx, cfg)`, apply overrides on top of the returned `*Bindings`, run `profiles.validate`, then populate the `*app.App`.
- [ ] 3.7 Tests in `profiles/lite_test.go`: every `Bindings` field non-nil after `Lite()`; compile-time interface satisfaction (`var _ harness.HarnessRunner = b.Runner`, etc.); `DataDir` is honored by the path-using stores (assert paths under temp dir).
- [ ] 3.8 Behavior parity test: pick three Layer 1 lite-store integration tests (one for `SessionStore`, one for `ArtifactStore`, one for `SecretStore`) and re-run them against `profiles.Lite()`-bound stores. Assert identical results.
- [ ] 3.9 Update `compose/compose_test.go`: add an end-to-end test that calls `compose.New(profiles.Lite(), compose.WithDataDir(t.TempDir()))` and asserts every accessor on the returned `*app.App` returns a non-nil value.
- [ ] 3.10 Run `go test ./compose/... ./profiles/... -race` and `go vet ./compose/... ./profiles/...`. Both green. Open the PR.

## 4. PR-3 — `profiles.Enterprise()` stub

- [ ] 4.1 Create `openharness/profiles/enterprise.go`. Define `Enterprise() Profile` returning a closure that immediately returns `nil, fmt.Errorf("enterprise profile requires Layer 5 to land; see docs/roadmap.md")`. Run `init()` to call `Register("enterprise", Enterprise())`.
- [ ] 4.2 Tests in `profiles/enterprise_test.go`: assert error contains `"layer 5"` (case-insensitive) and `"docs/roadmap.md"`; assert deterministic across 100 invocations; assert no goroutines or files leak (`goleak` if available, else `runtime.NumGoroutine` snapshot).
- [ ] 4.3 Update `compose/compose_test.go` with a test that `compose.New(profiles.Enterprise(), ...)` returns a nil `*app.App` and a non-nil error.
- [ ] 4.4 Add a `docs/profiles.md` first draft listing `lite` (live) and `enterprise` (stub pending Layer 5). Cross-link to `docs/architecture.md` and `docs/roadmap.md`.
- [ ] 4.5 Run `go test ./profiles/... -race`. Open the PR.

## 5. PR-4 — YAML loader + driver registries

- [ ] 5.1 Create `openharness/compose/drivers/` package with one driver registry per interface: `RegisterRunner(name, factory)`, `RegisterSessionStore(name, factory)`, etc. Each factory has signature `func(opts map[string]any) (harness.<Interface>, error)`. Re-registering an existing name returns `ErrDriverExists` and leaves the prior factory.
- [ ] 5.2 Register Lite drivers under the name `"lite"` for each of the eight interfaces, calling the same constructors `profiles.Lite()` uses.
- [ ] 5.3 Define the `openharness.yaml` schema as a Go struct in `compose/config.go`:
  - `version: int` (required, must equal `1`)
  - `profile: string` (required, resolves through `profiles.Lookup`)
  - `data_dir`, `edition`, `default_provider` (optional; mirror `compose.With*` options)
  - `runner`, `session_store`, `secret_store`, `artifact_store`, `tool_registry`, `channel_router`, `agent_store`, `skill_store` (each optional; each is `{driver: string, options: map}`)
  - `tools: []ToolRegistration` (optional; one entry per `Tools().Register` call)
  - `agents: []AgentSeed` (optional; one entry per `Agents().Seed` call)
- [ ] 5.4 Implement `compose.LoadFromFile(path string) (*app.App, error)`. Parse YAML (or JSON, sniffed by file extension), reject unknown top-level keys with line-numbered error, look up profile by name, resolve each interface override through the driver registry, build the equivalent `Option` slice, and call `compose.New`.
- [ ] 5.5 Implement post-composition tool/agent registration from the YAML's `tools` and `agents` arrays. Each registration emits a clear error on failure citing the YAML line.
- [ ] 5.6 Tests in `compose/config_test.go`: round-trip programmatic ↔ YAML (build the same `*app.App` two ways, compare accessor outputs); unknown-key rejection; missing-version rejection; unknown-profile rejection (error message lists available profiles); unknown-driver rejection.
- [ ] 5.7 Add a fixture `compose/testdata/lite.openharness.yaml` exercising the full schema. Loader test ingests it and asserts the resulting `*app.App` matches the programmatic equivalent.
- [ ] 5.8 Run `go test ./compose/... ./profiles/... -race`. Open the PR.

## 6. PR-5 — Reference vertical adoption

- [ ] 6.1 Wait for Plan 8 (`examples/minimal-vertical/` + `/create-openharness-app` scaffolder) to land on openharness `main`.
- [ ] 6.2 In `examples/minimal-vertical/main.go`, replace the `app.NewApp(...)` call with `compose.New(profiles.Lite(), compose.WithDataDir(...), ...)`. Keep the rest of the file (tool/agent registration, `Run()`) unchanged.
- [ ] 6.3 Add `examples/minimal-vertical/openharness.yaml` next to `main.go` exercising the YAML loader path. Add a `make run-yaml` (or equivalent) that runs the example via `compose.LoadFromFile`.
- [ ] 6.4 Update the scaffolder skill (`skills/create-openharness-app/SKILL.md`) so newly scaffolded apps emit `compose.New(profiles.Lite(), ...)` and an `openharness.yaml` template. Document the choice in the SKILL.md's "Decisions" section.
- [ ] 6.5 Add a CI gate (`.github/workflows/scaffolder-ci.yml` or whatever Plan 8 created): run scaffolder → `go mod tidy` → `go build` → `go test`; assert the generated `main.go` contains `compose.New(profiles.Lite()` (string match).
- [ ] 6.6 Run the example end-to-end manually: `cd examples/minimal-vertical && go run .` then `go run . --config openharness.yaml`. Confirm both paths boot the agent loop without panic.

## 7. PR-6 — Documentation reframe

- [ ] 7.1 Rewrite the lead section of `docs/architecture.md` around composition: openharness as a platform-builder runtime; eight swappable interfaces; profiles as named presets; `compose.New` as the single entrypoint. Show a 10-line example.
- [ ] 7.2 Add a "§Composition" subsection to `docs/superpowers/specs/2026-04-10-openharness-extraction-design.md` (under §System Architecture). Cross-link to `openspec/changes/platform-builder-runtime/`. Note that this section is the canonical platform-builder framing; older "extracted from ghostfin" framing in the same doc gets a one-line preface that points readers here.
- [ ] 7.3 Update `docs/roadmap.md`: re-key Layer 3 (telemetry/cost/sessions concretes) as "binds into `profiles.Lite()` and `profiles.Enterprise()`". Re-key Layer 4 (app scaffold) as "Lite-profile lifecycle option". Re-key Layer 5 as "Enterprise-profile bindings replace the stub". Re-key Layer 6 (SDK) as "transports over an already-built `*app.App`".
- [ ] 7.4 Update `README.md` (top-level): replace any "framework extracted from ghostfin" framing with "platform for building agentic applications". Show the `compose.New(profiles.Lite(), ...)` quickstart.
- [ ] 7.5 Polish `docs/profiles.md` (started in PR-3) to be the canonical reference for the profile contract, the built-in profiles, and how to register custom profiles. Include the `profiles.LiteBindings` compose-and-patch pattern.
- [ ] 7.6 Add `docs/openharness-yaml.md` documenting the `openharness.yaml` schema with examples for each interface override and tool/agent registration.

## 8. Verification

- [ ] 8.1 Full test sweep: `go test ./... -race` and `go vet ./...` from repo root. Zero failures, zero `vet` complaints.
- [ ] 8.2 Layer 1 parity gate: re-run the Layer 1 lite-store integration tests against `profiles.Lite()`-bound stores. Identical pass set vs the existing manual-wiring tests.
- [ ] 8.3 Scaffolder end-to-end: in a scratch directory, run `/create-openharness-app` against a fresh test vertical name. Verify the generated module passes `go mod tidy && go build && go test ./...` and that its `main.go` uses `compose.New`.
- [ ] 8.4 Manual smoke: `cd examples/minimal-vertical && go run .` boots the agent chat loop. Send one message, see one streamed response. Stop with Ctrl+C; confirm no goroutine leaks (use `runtime.NumGoroutine` printout if no goleak).
- [ ] 8.5 YAML smoke: same example launched via `go run . --config openharness.yaml` produces identical observable behavior.
- [ ] 8.6 Backward-compat smoke: a pre-change consumer of `app.NewApp(AppConfig{})` (synthesized in a test fixture) still compiles and runs against the new module.
- [ ] 8.7 Update this task list as each PR merges. The change is complete when PR-6 lands and verification §8 passes.
- [ ] 8.8 OpenSpec archive: once all checkboxes here are checked and PRs merged, run `/opsx:archive` to roll `specs/app-composition` and `specs/runtime-profiles` into `openspec/specs/` permanently.

## 9. Follow-ups (captured as separate OpenSpec changes when ready, not blocking this change)

- [ ] 9.1 Replace `profiles.Enterprise()` stub with real bindings once Layer 5 lands.
- [ ] 9.2 Add `Bindings.TraceRecorder` and `Bindings.SpanSink` once Layer 3 telemetry refactor lands (per design.md Open Question 1).
- [ ] 9.3 Decide on global vs per-invocation driver registries (per design.md Open Question 2). Currently global; `compose.NewIsolated(...)` test-only constructor TBD.
- [ ] 9.4 Profile composition / inheritance API (per design.md Open Question 3). Currently `LiteBindings` compose-and-patch; revisit if real-world friction emerges.
- [ ] 9.5 `openharness.yaml` schema versioning policy (per design.md Open Question 4). Currently `version: 1` only; document `version: 2` policy when first break occurs.
- [ ] 9.6 Ghostfin migration: after Plan 7 lands, propose a follow-up change to switch ghostfin's `desktop/main.go` from `app.NewApp(...)` to `compose.New(profiles.Lite(), ...)`.

## 10. OSS pathway research (2026-09-05; advisory, no implementation scope change)

- [x] 10.1 Compare current OpenHarness and Kiteframe with kagent, Google AX and sandbox providers; record source baselines, implemented/planned boundaries and a proposed incremental pathway in `docs/research/2026-09-05-declarative-agents-and-sandboxes.md`.
- [ ] 10.2 Decide whether to adopt the proposed goal-kernel / harness / sandbox ownership split and first backend experiment. Capture accepted changes in a subsequent OpenSpec proposal before implementation; existing Plan 6 prerequisites and unchecked tasks remain in force.
