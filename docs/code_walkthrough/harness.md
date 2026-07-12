# harness

First-depth walkthrough of `harness/`.

This is the centerpiece of openharness's "pluggable runtime" framing. The package is interface-only at depth 1 (plus a runner registry and a polling event-stream adapter); concrete implementations live in `harness/lite/` and need a depth-2 walkthrough.

## Package summary

`harness` defines the boundary between a gateway/orchestrator and any backend that runs agents. It declares **eight interfaces** that together describe a multi-tenant agentic runtime: dispatch, durable session log, secrets, artifacts, tool catalog, channel routing, agent definition store, skill store. It also ships an `EventStream` iterator and a `RunnerRegistry` for selecting a backend by name.

**Importers (today):** `harness/lite` (provides concrete impls); `providers/secret.go` consumes `harness.SecretStore` ([`providers/secret.go`](../../providers/secret.go)).

## Files at depth 1

- `harness/interfaces.go` (84 lines) — Eight boundary interfaces.
- `harness/types.go` (256 lines) — Domain data types, event-type constants, sandbox policy.
- `harness/errors.go` (14 lines) — Eight sentinel errors.
- `harness/runner_registry.go` (40 lines) — Thread-safe `name → HarnessRunner` map.
- `harness/stream_store.go` (89 lines) — `StoreBackedEventStream` polling adapter for remote runners.
- `harness/interfaces_test.go` (181 lines) — Compile-time interface assertions, event-type sanity, RunConfig field checks.
- `harness/runner_registry_test.go` (47 lines) — Register/Get/Not-Found.
- `harness/stream_store_test.go` (145 lines) — Polling, terminal-event detection, Close → EOF.

## Subdirectories at depth 1

`harness/lite/` (NOT walked here; gets its own depth-2 doc later). Filenames observed:

- `agent.go`, `artifact.go`, `channel.go`, `registry.go`, `runner.go`, `secret.go`, `skill.go` — concrete implementations, one per harness interface.
- `agent_test.go`, `artifact_test.go`, `channel_test.go`, `registry_test.go`, `runner_test.go`, `secret_test.go`, `skill_test.go`, `satisfy_test.go` — tests including a compile-time `satisfy_test.go` likely asserting the interfaces are implemented.

## Public surface

### Interfaces

Grouped by responsibility:

**Execution & dispatch**

- `HarnessRunner` — [`interfaces.go:12-18`](../../harness/interfaces.go) — `Dispatch(ctx, RunConfig) (EventStream, error)`, `Resume(ctx, sessionID) (EventStream, error)`, `Interrupt(ctx, runID, Event) error`, `Cancel(ctx, runID) error`, `Status(ctx, runID) (RunStatus, error)`.

**Durable storage**

- `SessionStore` — [`interfaces.go:21-28`](../../harness/interfaces.go) — `CreateSession`, `AppendEvent`, `GetEvents(fromPos)`, `GetSession`, `SetCheckpoint`, `ListSessions`. Append-only event log with checkpointing.
- `ArtifactStore` — [`interfaces.go:38-43`](../../harness/interfaces.go) — `Put`, `Get`, `List`, `Presign`. Object storage with presigned URLs.
- `SkillStore` — [`interfaces.go:77-83`](../../harness/interfaces.go) — `ListPlatform`, `ListTenant`, `Resolve`, `Upload`, `Delete`.

**Catalog & config**

- `ToolRegistry` — [`interfaces.go:46-54`](../../harness/interfaces.go) — Global + per-tenant overlay; `Resolve`, `RegisterTool`, `UpdateTool`, `DeleteTool`, `VerifyTool`.
- `AgentStore` — [`interfaces.go:66-74`](../../harness/interfaces.go) — Versioned `AgentDefinition` CRUD with optimistic locking; `Archive`, `Clone`, `ListVersions`.

**Integration**

- `SecretStore` — [`interfaces.go:31-35`](../../harness/interfaces.go) — `GetCredentials`, `MintScopedToken(ttl)`, `StoreCredentials`.
- `ChannelRouter` — [`interfaces.go:57-63`](../../harness/interfaces.go) — `HandleInbound`, `SendOutbound`, `RegisterWebhook`, `UnregisterWebhook`, `ListChannels`.

**Streaming**

- `EventStream` — [`types.go:60-63`](../../harness/types.go) — `Next() (Event, error)`, `Close() error`. Iterator from `Dispatch`/`Resume`.

### Types (domain entities)

- Run lifecycle: `Event` ([`types.go:23-32`](../../harness/types.go)), `RunConfig` ([`types.go:35-47`](../../harness/types.go)), `RunStatus` ([`types.go:50-57`](../../harness/types.go)), `Snapshot` ([`types.go:238-245`](../../harness/types.go)).
- Agent config: `AgentDefinition` ([`types.go:66-88`](../../harness/types.go)), `AgentPatch` ([`types.go:109-120`](../../harness/types.go)), `PersonaFiles` ([`types.go:91-97`](../../harness/types.go)), `MCPServerConfig` ([`types.go:100-105`](../../harness/types.go)).
- Tools: `ToolDefinition` ([`types.go:123-136`](../../harness/types.go)), `VerifyResult` ([`types.go:139-143`](../../harness/types.go)).
- Skills: `SkillMeta` ([`types.go:146-156`](../../harness/types.go)), `Skill` ([`types.go:159-163`](../../harness/types.go)), `SkillRef` ([`types.go:167-170`](../../harness/types.go)).
- Sessions: `Session` ([`types.go:180-187`](../../harness/types.go)), `SessionMeta` ([`types.go:173-177`](../../harness/types.go)).
- Channels: `Message` ([`types.go:190-196`](../../harness/types.go)), `InboundResult` ([`types.go:199-203`](../../harness/types.go)), `ChannelConfig` ([`types.go:206-211`](../../harness/types.go)), `ChannelInfo` ([`types.go:214-220`](../../harness/types.go)).
- Artifacts: `ArtifactMeta` ([`types.go:223-228`](../../harness/types.go)).
- Pagination: `ListOpts` ([`types.go:231-235`](../../harness/types.go)).
- Sandbox: `SandboxPolicy` ([`types.go:250-255`](../../harness/types.go)) — `AllowedPaths`, `DeniedCommands`, `TimeoutSeconds`, `NetworkAccess`. Nil = unrestricted.

### Functions

- `NewRunnerRegistry()` — [`runner_registry.go:17`](../../harness/runner_registry.go).
- `(*RunnerRegistry).Register(name, HarnessRunner)` — [`runner_registry.go:24`](../../harness/runner_registry.go).
- `(*RunnerRegistry).Get(name) (HarnessRunner, error)` — [`runner_registry.go:32`](../../harness/runner_registry.go).
- `NewStoreBackedEventStream(ctx, SessionStore, sessionID)` — [`stream_store.go:29`](../../harness/stream_store.go).
- `(*StoreBackedEventStream).Next() (Event, error)` — [`stream_store.go:44`](../../harness/stream_store.go) — Polls at `defaultPollInterval` (200ms).
- `(*StoreBackedEventStream).Close() error` — [`stream_store.go:83`](../../harness/stream_store.go).

### Errors

[`errors.go:5-14`](../../harness/errors.go) — `ErrNotFound`, `ErrConflict`, `ErrArchived`, `ErrUnauthorized`, `ErrRunNotActive`, `ErrTokenExpired`, `ErrVerifyFailed`, `ErrSkillNotFound`.

### Constants (event types)

[`types.go:9-20`](../../harness/types.go) — `EventRunStarted`, `EventChunk`, `EventToolCall`, `EventToolResult`, `EventRunCompleted`, `EventError`, `EventAskUser`, `EventActivity`, `EventModeSwitch`, `EventCheckpoint`.

## Pluggability hooks

This is where the "pluggable runtime" claim is spent. Each interface is a swap point; today the only impl set is `harness/lite/`.

- **`HarnessRunner` + `RunnerRegistry`**: register many runners by name; gateway dispatches by name. `AgentDefinition.Runner` and `FallbackRunner` ([`types.go:79-80`](../../harness/types.go)) let agent configs choose the backend. **Concrete impl in this depth: none.** Lite runner is in `harness/lite/runner.go`.
- **`SessionStore`**: `StoreBackedEventStream` ([`stream_store.go`](../../harness/stream_store.go)) is the canonical adapter for remote runners — they write events to a shared store; the gateway polls. This is the explicit "K8s/Lambda" pattern referenced in the interface doc-comment.
- **`SecretStore`** consumed by `providers.ResolveAPIKeyWithStore` ([`providers/secret.go`](../../providers/secret.go)) — a real cross-package coupling that proves the seam is load-bearing.
- **Other interfaces** (`ArtifactStore`, `ToolRegistry`, `ChannelRouter`, `AgentStore`, `SkillStore`) have no factory or registry — they are wired by composition (DI) wherever the gateway is assembled. That assembly site is not visible at depth 1; reviewer should locate it.

## Pluggability evidence summary

Direct anchors a reviewer should jump to:

1. [`interfaces.go:12-18`](../../harness/interfaces.go) — `HarnessRunner` interface.
2. [`runner_registry.go:11-40`](../../harness/runner_registry.go) — Named-runner registry.
3. [`interfaces.go:21-28`](../../harness/interfaces.go) — `SessionStore` interface.
4. [`stream_store.go:15-39`](../../harness/stream_store.go) — `StoreBackedEventStream` polling adapter for remote runners.
5. [`interfaces.go:46-54`](../../harness/interfaces.go) — `ToolRegistry` global + tenant overlay.
6. [`types.go:79-80`](../../harness/types.go) — `AgentDefinition.Runner` / `FallbackRunner` for per-agent backend selection.
7. [`types.go:250-255`](../../harness/types.go) — `SandboxPolicy` (constraint hook for tool execution).
8. [`interfaces_test.go`](../../harness/interfaces_test.go) — compile-time assertions for interface contracts.

## External deps used

stdlib only at depth 1: `context`, `encoding/json`, `io`, `time`, `errors`, `fmt`, `sync`.

## Notable tests

- `TestRunnerRegistry_RegisterAndGet` — [`runner_registry_test.go:22`](../../harness/runner_registry_test.go).
- `TestRunnerRegistry_GetUnregisteredReturnsError` — [`runner_registry_test.go:37`](../../harness/runner_registry_test.go) — Wraps `ErrNotFound`.
- `TestStoreBackedEventStreamNext` — [`stream_store_test.go:53`](../../harness/stream_store_test.go) — Three-event happy path.
- `TestStoreBackedEventStreamClose` — [`stream_store_test.go:96`](../../harness/stream_store_test.go) — `Close` → `Next` returns `io.EOF`.
- `TestStoreBackedEventStreamTerminalEvent` — [`stream_store_test.go:116`](../../harness/stream_store_test.go) — `EventRunCompleted` auto-closes.
- `interfaces_test.go` — compile-time `var _ Iface = (*stub)(nil)` for all eight interfaces; spot-checks `RunConfig`, `AgentDefinition.Runner`/`FallbackRunner`, sandbox-nil-is-unrestricted.

## Open questions / gaps

These belong to a depth-2 walkthrough of `harness/lite/`:

1. How does `LiteRunner` actually execute the agent loop? Embedded goroutine; does it use `tools/` directly?
2. Does `harness/lite/` provide a session store that satisfies `harness.SessionStore`? If so, is it filesystem- or in-memory-backed? How does it relate to the parallel `sessions/` package (see [sessions](./sessions.md))?
3. Where is the wiring point — the place that constructs all eight subsystems and hands them to a gateway? At depth 1 there is no such constructor.
4. `ToolRegistry.VerifyTool` semantics (endpoint reachability? schema validation?) — defined only by interface; proof is in lite impl.
5. `SandboxPolicy` enforcement: which runners actually consult it? At depth 1 it is purely a data type.
6. `defaultPollInterval = 200ms` ([`stream_store.go:10`](../../harness/stream_store.go)) — not configurable; remote runners may want backoff.
7. `MintScopedToken` — what does "scoped" mean? Probably tenant-bound TTL; confirm via lite secret impl.
8. Skills' `SkillRef.Version == 0 ⇒ latest` semantics ([`types.go:167-170`](../../harness/types.go)) — verify in `lite/skill.go`.
