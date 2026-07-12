## ADDED Requirements

### Requirement: Profile contract

The framework SHALL define `profiles.Profile` as a function type `func(ctx context.Context, cfg ProfileConfig) (*Bindings, error)` where `Bindings` is a struct holding one nullable field per harness interface. A profile MUST return a `*Bindings` with **every** interface field non-nil on success. A profile MAY return an error (e.g., when a required precondition like a Postgres URL is missing); on error it MUST NOT leak partially-constructed resources. `ProfileConfig` MUST carry at least: `DataDir string`, `Edition edition.Edition`, `DefaultProvider string`, plus a free-form `Options map[string]any` for profile-specific keys.

#### Scenario: Successful profile binds every interface

- **WHEN** `profiles.Lite()` is invoked with a valid `ProfileConfig{DataDir: "/tmp/app", Edition: edition.Lite}`
- **THEN** the returned `*Bindings` has non-nil `Runner`, `SessionStore`, `SecretStore`, `ArtifactStore`, `ToolRegistry`, `ChannelRouter`, `AgentStore`, `SkillStore`
- **AND** each binding satisfies the corresponding `harness.*` interface (compile-time check via `_ harness.HarnessRunner = bindings.Runner`)

#### Scenario: Profile reports missing precondition

- **WHEN** `profiles.Enterprise()` is invoked without the required Postgres connection string in `ProfileConfig.Options`
- **THEN** the profile returns a nil `*Bindings` and an error of the form `enterprise profile requires <key>; got none`
- **AND** the error names the specific missing key so the caller can fix their config

### Requirement: Built-in Lite profile

The framework SHALL ship `profiles.Lite()` that binds every interface to the existing `harness/lite/*` implementation: `lite.LiteRunner`, `lite.LiteSessionStore`, `lite.LiteSecretStore`, `lite.LiteArtifactStore`, `lite.LiteToolRegistry`, `lite.LiteChannelRouter`, `lite.LiteAgentStore`, `lite.LiteSkillStore`. Behavior MUST match what ghostfin currently gets from manual wiring of those types — Lite profile is a re-export under named bindings, not a re-implementation. Default storage paths SHALL be derived from `ProfileConfig.DataDir`.

#### Scenario: Lite profile is behaviorally identical to manual wiring

- **WHEN** the Layer 1 lite-store integration tests are run against (a) manually-constructed Lite stores and (b) `profiles.Lite()`-bound stores
- **THEN** all tests pass identically against both
- **AND** no Lite store implementation gains a code path specific to the profile-driven construction (the profile is wiring code, not new behavior)

#### Scenario: DataDir flows to all path-bearing stores

- **WHEN** `profiles.Lite()` is invoked with `ProfileConfig{DataDir: "/var/app"}`
- **THEN** `LiteSessionStore` writes its JSONL at a subpath under `/var/app`
- **AND** `LiteArtifactStore` writes blobs under `/var/app`
- **AND** `LiteSkillStore` reads/writes manifests under `/var/app`
- **AND** stores that don't use the filesystem (e.g. `LiteToolRegistry`, in-memory) ignore `DataDir`

### Requirement: Stub Enterprise profile

The framework SHALL ship `profiles.Enterprise()` immediately on this change landing, even though Layer 5's enterprise impls do not yet exist. The stub MUST return a non-nil error from its binding function with a message citing the unbuilt dependency (e.g. `"enterprise profile requires Layer 5 to land; tracked at openharness-pol.5"`). This guarantees the profile name is reserved and that calling it produces a deterministic, debuggable failure rather than a registry miss. Once Layer 5 lands, the stub is replaced by a real binding function in the same package without API changes.

#### Scenario: Enterprise stub fails clearly today

- **WHEN** `compose.New(profiles.Enterprise(), ...)` is invoked at any point before Layer 5 ships
- **THEN** `compose.New` returns a nil `*app.App` and an error containing `"layer 5"` or `"enterprise"` and a tracking link/issue ID
- **AND** the error is reproducible across repeated invocations (no flakiness)

#### Scenario: Enterprise replacement is non-breaking

- **WHEN** Layer 5 lands and replaces the stub with a real binding
- **THEN** existing callsites of `compose.New(profiles.Enterprise(), ...)` start succeeding without source changes
- **AND** the `profiles.Enterprise` symbol's signature is unchanged (still `Profile`)

### Requirement: Custom profile registration

The framework SHALL allow verticals and operators to define their own profiles and register them by name via `profiles.Register(name string, p Profile) error`. Registered profiles SHALL be addressable from the YAML loader by their registered name. Re-registering an already-registered name MUST return a `ErrProfileExists` sentinel error rather than silently overwriting. Unregistered names referenced from YAML MUST cause a load failure with the available profile names listed in the error message.

#### Scenario: Register and load a custom profile

- **GIVEN** a vertical defines `func myLiteWithS3(ctx, cfg) (*Bindings, error) { ... }`
- **WHEN** the vertical calls `profiles.Register("my-lite-s3", myLiteWithS3)` at startup
- **AND** loads `openharness.yaml` containing `profile: my-lite-s3`
- **THEN** the loader resolves the profile and binds via `myLiteWithS3`
- **AND** the resulting `*app.App` reflects whatever bindings `myLiteWithS3` set

#### Scenario: Conflicting registration is rejected

- **WHEN** a caller registers a profile under a name that is already registered (e.g. `lite`)
- **THEN** `profiles.Register` returns `ErrProfileExists` wrapping the duplicated name
- **AND** the existing registration is unchanged (no partial overwrite)

#### Scenario: Unknown profile name in YAML

- **WHEN** `compose.LoadFromFile` reads a config containing `profile: nonexistent`
- **THEN** the loader returns an error of the form `unknown profile "nonexistent"; available: lite, enterprise, my-lite-s3`
- **AND** no `*app.App` is returned

### Requirement: Profile is the runner-pluggability surface

The framework SHALL document `profiles.Profile` (and the `Bindings.Runner` field specifically) as the place where alternate `HarnessRunner` implementations are introduced. Profile authors writing a profile that binds a non-Lite runner (e.g. K8s Jobs, Lambda, future HTTP-driven worker) SHALL set `Bindings.Runner` to their implementation; `compose.New` consumes whatever the profile returns. This formalizes that runner pluggability is a first-class composition concern, not an after-thought reached via `harness.RunnerRegistry` alone. The existing `RunnerRegistry` continues to serve dispatch-by-name lookups (e.g. an `AgentDefinition` declares `runner: "lambda"`); profiles populate the registry during binding.

#### Scenario: Profile sets Bindings.Runner

- **WHEN** a custom profile constructs a `*K8sJobRunner` and assigns it to `Bindings.Runner`
- **THEN** `compose.New` returns an `*app.App` whose `Runner()` accessor returns that `*K8sJobRunner`
- **AND** the `RunnerRegistry` (also held on the App) contains the K8s runner registered under whatever name the profile chose

#### Scenario: RunnerRegistry remains the dispatch surface

- **WHEN** an `AgentDefinition` is created with `Runner: "lambda"` and the profile registered a `LambdaRunner` under the name `"lambda"`
- **THEN** dispatching the agent resolves through `RunnerRegistry.Get("lambda")` to the registered `LambdaRunner`
- **AND** runners not in the registry surface a clear error at dispatch time, not at composition time
