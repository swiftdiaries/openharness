## ADDED Requirements

### Requirement: Programmatic composition entrypoint

The framework SHALL provide a `compose.New(profile, opts ...Option) (*app.App, error)` function that returns a fully bound `*app.App` value with all eight harness interfaces (`HarnessRunner`, `SessionStore`, `SecretStore`, `ArtifactStore`, `ToolRegistry`, `ChannelRouter`, `AgentStore`, `SkillStore`) populated. Callers MUST be able to obtain a runnable app from a single call without manually wiring stores. The returned `*app.App` MUST be the same value type produced by Plan 6's `app.NewApp(AppConfig)` so downstream registration calls (`Tools()`, `Agents()`, `Telemetry()`, `Frontend()`, `Run()`) work identically regardless of construction path.

#### Scenario: Lite profile produces a runnable app

- **WHEN** a caller invokes `compose.New(profiles.Lite(), compose.WithDataDir("/tmp/app"))`
- **THEN** the call returns a non-nil `*app.App` and a nil error
- **AND** every accessor (`app.Runner()`, `app.SessionStore()`, `app.SecretStore()`, `app.ArtifactStore()`, `app.ToolRegistry()`, `app.ChannelRouter()`, `app.AgentStore()`, `app.SkillStore()`) returns a non-nil implementation
- **AND** `app.Run(ctx)` starts without panicking when no tools or agents have been registered (`Run` returns a clear error message naming the missing registration, not a nil-pointer panic)

#### Scenario: Profile error is surfaced

- **WHEN** the supplied profile's binding function returns an error (e.g. `profiles.Enterprise()` before Layer 5 lands)
- **THEN** `compose.New` returns a nil `*app.App` and the profile's error wrapped with sufficient context to identify which interface binding failed
- **AND** no goroutines, file handles, or open connections are leaked from partially-bound stores

### Requirement: Per-interface override options

The `compose` package SHALL expose `With*` option functions for each of the eight harness interfaces — `WithRunner`, `WithSessionStore`, `WithSecretStore`, `WithArtifactStore`, `WithToolRegistry`, `WithChannelRouter`, `WithAgentStore`, `WithSkillStore`. An override option MUST take precedence over the profile-supplied default for that interface. Multiple options for the same interface MUST cause `compose.New` to return an error rather than silently taking the last one.

#### Scenario: Override replaces profile default

- **WHEN** a caller invokes `compose.New(profiles.Lite(), compose.WithSecretStore(myVaultStore))`
- **THEN** `app.SecretStore()` returns `myVaultStore`
- **AND** the other seven interfaces are still bound by `profiles.Lite()`'s defaults

#### Scenario: Conflicting overrides are rejected

- **WHEN** a caller invokes `compose.New(profiles.Lite(), compose.WithSecretStore(a), compose.WithSecretStore(b))`
- **THEN** `compose.New` returns a nil `*app.App` and an error naming the duplicated option
- **AND** the error message identifies the option name (`WithSecretStore`) so the caller can correct their code

### Requirement: Declarative composition via `openharness.yaml`

The framework SHALL support loading a composition definition from a YAML file at `openharness.yaml` (or a path supplied to `compose.LoadFromFile(path)`). The file MUST select a profile by name, MAY override individual interface bindings by named driver (e.g. `secret_store: vault`), and MAY list tool/agent/channel registrations. The loader MUST produce the same `*app.App` value as the equivalent programmatic invocation. JSON files with the same schema MUST be accepted via the same loader. Unknown top-level keys MUST cause a load failure with a clear error message; this prevents typos from silently dropping configuration.

#### Scenario: Load Lite profile from YAML

- **WHEN** a caller invokes `compose.LoadFromFile("openharness.yaml")` against a file containing `profile: lite` and `data_dir: /tmp/app`
- **THEN** the loader returns a `*app.App` equivalent to `compose.New(profiles.Lite(), compose.WithDataDir("/tmp/app"))`
- **AND** every interface accessor returns the same driver type (e.g. `*lite.LiteSessionStore`) as the programmatic path

#### Scenario: Override drivers via YAML

- **WHEN** a caller loads a YAML file with `profile: lite` and `secret_store: { driver: keychain, options: { service_name: "test" } }`
- **THEN** `app.SecretStore()` returns the keychain driver instance configured with `service_name="test"`
- **AND** the other seven interfaces remain bound by `profiles.Lite()` defaults

#### Scenario: Unknown key fails loudly

- **WHEN** a caller loads a YAML file containing a top-level key not declared in the schema (e.g. `databse: foo`)
- **THEN** the loader returns a nil `*app.App` and an error citing the unrecognized key and its file location (line number, file path)
- **AND** the error does not mention any successfully-parsed bindings

### Requirement: Composition is the only public path to App construction

After this change lands, the documented public surface for assembling an `*app.App` is `compose.New(...)` and `compose.LoadFromFile(...)`. Plan 6's `app.NewApp(AppConfig)` constructor is downgraded to an unexported helper (or kept exported but documented as "low-level; prefer compose.New"). Direct instantiation of `app.App{}` from outside the `compose` and `app` packages SHALL be unsupported. This freezes the platform-builder entrypoint and prevents the proliferation of bespoke store-wiring code in verticals.

#### Scenario: Direct App construction is discouraged

- **WHEN** a caller writes `app.NewApp(AppConfig{})` in their main package
- **THEN** their code still compiles (Plan 6 export retained for backward compatibility through 0.x)
- **AND** the godoc for `app.NewApp` documents `compose.New` as the preferred entrypoint and links to a migration example

#### Scenario: Compose is the documented path

- **WHEN** the openharness `docs/architecture.md` and the scaffolder-generated `main.go` show how to assemble an app
- **THEN** they uniformly use `compose.New(profiles.Lite(), ...)` rather than `app.NewApp(...)`
- **AND** the reference vertical at `examples/minimal-vertical/` exercises `compose.New` end-to-end (CI-verified)
