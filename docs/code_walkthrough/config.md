# config

First-depth walkthrough of `config/`.

## Package summary

`config` provides JSON-on-disk application configuration: providers, MCP servers, agents, channels, cost limits, workspace, compaction, plus a fold of "enterprise" fields (firm/user/PG DSN). It also implements an upward-walking discovery for instruction markdown files.

**Importers (today):** none discovered in the rest of the repo at depth 1. The package compiles in isolation; downstream wiring (gateway, agent loop) has not yet adopted it.

## Files at depth 1

- `config/config.go` (157 lines) — All struct definitions and `Default()` / `ResolveModel()` factories.
- `config/discovery.go` (56 lines) — `DiscoverInstructionFiles(startDir)` walks up the directory tree.
- `config/store.go` (67 lines) — `Load(path)` / `Save(path, cfg)` JSON persistence with atomic temp-then-rename.
- `config/config_test.go` (130 lines) — Defaults, save/reload roundtrip, model resolution.
- `config/discovery_test.go` (52 lines) — Upward traversal, truncation at 4000 chars, no-files-found.
- `config/store_test.go` (105 lines) — Atomic write, error preservation, 0o600 file perms.

## Subdirectories at depth 1

None.

## Public surface

### Types

- `AppConfig` — [`config.go:9-26`](../../config/config.go) — Top-level root.
- `WorkspaceConfig` — [`config.go:30-32`](../../config/config.go).
- `GatewayConfig` — [`config.go:35-38`](../../config/config.go).
- `ProviderConfig` — [`config.go:41-48`](../../config/config.go).
- `MCPServerConfig` — [`config.go:51-66`](../../config/config.go) — Including OAuth client fields.
- `AgentOverride` — [`config.go:69-74`](../../config/config.go).
- `ChannelConfig` — [`config.go:77-81`](../../config/config.go).
- `CostConfig` — [`config.go:84-87`](../../config/config.go).
- `CompactionSettings` — [`config.go:90-93`](../../config/config.go).
- `InstructionFile` — [`discovery.go:13-16`](../../config/discovery.go).

### Functions

- `Default() AppConfig` — [`config.go:102`](../../config/config.go).
- `DefaultWorkspacePath() string` — [`config.go:97`](../../config/config.go).
- `ResolveModel(agentName, cfg) string` — [`config.go:150`](../../config/config.go).
- `DiscoverInstructionFiles(startDir) []InstructionFile` — [`discovery.go:18`](../../config/discovery.go).
- `Load(path) (AppConfig, error)` — [`store.go:11`](../../config/store.go).
- `Save(path, cfg) error` — [`store.go:31`](../../config/store.go).

### Constants

- `MaxInstructionFileChars = 4000` — [`discovery.go:9`](../../config/discovery.go).
- `MaxTotalInstructionChars = 12000` — [`discovery.go:10`](../../config/discovery.go).

## Pluggability hooks

The `AppConfig` struct itself is the extension surface — verticals append to slices:

- `Providers []ProviderConfig` — [`config.go:12`](../../config/config.go).
- `MCPServers []MCPServerConfig` — [`config.go:13`](../../config/config.go).
- `Agents []AgentOverride` — [`config.go:14`](../../config/config.go).
- `Channels []ChannelConfig` — [`config.go:15`](../../config/config.go).
- `Compaction CompactionSettings` — [`config.go:17`](../../config/config.go).

There is no interface or registry — extension is purely declarative via JSON.

## Vertical-specific bleed-through

This is the most striking gap for the "framework" framing. Several ghostfin-specific defaults leaked into the framework package:

1. **`DefaultWorkspacePath()` hardcodes `~/.ghostfin/workspace`** — [`config.go:97-100`](../../config/config.go).
2. **`DiscoverInstructionFiles` searches for `GHOSTFIN.md`, `.ghostfin/GHOSTFIN.md`, `GHOSTFIN.local.md`** — [`discovery.go:19`](../../config/discovery.go).
3. **`MgmtServerURL` doc example is `http://mgmt.ghostfin.io:8080`** — [`config.go:22`](../../config/config.go).
4. **`Default().Agents` ships ghostfin personas:** `harness`, `analyst`, `scribe`, `executive-assistant`, `fund-analyst` — [`config.go:127-133`](../../config/config.go). `scribe` and `fund-analyst` are vertical concepts; `harness` is the only generic name.
5. **`WorkspaceConfig.Path` doc-comment** says `default: ~/.ghostfin/workspace/` — [`config.go:31`](../../config/config.go).

A reviewer evaluating "pluggable runtime" should flag this: a framework package should not ship vertical defaults. Either remove or parameterize via `DiscoveryConfig{ Names []string }` and per-vertical overlays.

## External deps used

stdlib only: `os`, `path/filepath`, `encoding/json`, `errors`, `bufio`-equivalent via `os.ReadFile`.

## Notable tests

- `TestLoadDefaultConfig` — [`config_test.go:9`](../../config/config_test.go) — Pins the default ports, providers, MCP catalog, daily/monthly cost limits, default model name, and the count of 5 default agents.
- `TestSaveAndReload` — [`config_test.go:48`](../../config/config_test.go) — Roundtrip with an added `github` MCP server.
- `TestResolveModel_*` (4 tests) — [`config_test.go:100-130`](../../config/config_test.go) — Per-agent override, harness override, fallback to default, unknown agent.
- `TestDiscoverInstructionFiles*` — [`discovery_test.go`](../../config/discovery_test.go) — Upward traversal order, size-limit truncation, no-files-found.
- `TestSaveAtomicWrite` — [`store_test.go:10`](../../config/store_test.go) — No temp debris on success.
- `TestSaveOverwritePreservesOnError` — [`store_test.go:44`](../../config/store_test.go) — Original preserved when temp create fails.
- `TestSaveFilePermissions` — [`store_test.go:90`](../../config/store_test.go) — Pins 0o600 mode.

## Open questions / gaps

1. **Framework-vs-vertical naming.** Either rename `GHOSTFIN.md` → `OPENHARNESS.md` and `~/.ghostfin/` → `~/.openharness/`, or make these injectable.
2. **`Default().Agents` is opinionated.** Should default to `nil`; verticals populate.
3. **No integration with `harness/`.** `AppConfig.Agents` doesn't feed into `harness.AgentStore`. A reviewer should ask how config-time agent definitions become runtime entities.
4. **Enterprise fields untested.** `Edition`, `MgmtServerURL`, `FirmID`, `UserID`, `PGDSN` are declared but no test exercises them.
5. **Truncation edge case.** If the first discovered file is ≤ 4000 chars but ≤ 12000 cap is hit by the next file, the inner loop breaks before adding it but the outer loop still advances dir — correct, but worth a test case.
