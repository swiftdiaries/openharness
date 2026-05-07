# sessions

First-depth walkthrough of `sessions/`.

## Package summary

`sessions` provides a JSONL-on-disk session log: each session is a single `.jsonl` file with a `meta` line followed by one `message` line per turn. Token usage can be attached per-message.

**Crucial:** this is **not** an implementation of `harness.SessionStore`. The two operate at different abstraction levels:

| | `sessions.Store` | `harness.SessionStore` |
|---|---|---|
| Method receivers | None take `context.Context` | All take `context.Context` |
| Multi-tenant | No tenant ID | Tenant-scoped |
| Event model | `Message` (role/content) | `Event` (typed: chunk, tool.call, etc.) |
| Checkpointing | Absent | `SetCheckpoint(pos)` |
| Resume | Absent | Implicit via `GetEvents(fromPos)` |
| Concurrency | Single-process JSONL | Designed for shared store / polling |

There is a compile-time `var _ interface{...} = (*Store)(nil)` ([`store.go:58-66`](../../sessions/store.go)) that asserts an inline narrow interface, but no part of the codebase uses that interface as an abstraction.

**Importers (today):** none in the rest of the repo.

## Files at depth 1

- `sessions/store.go` (175 lines).
- `sessions/store_test.go` (72 lines).

## Public surface

- `Session` — [`store.go:16-20`](../../sessions/store.go) — ID, AgentName, CreatedAt.
- `Message` — [`store.go:22-30`](../../sessions/store.go) — Type, Role, Content, ToolCalls, ToolCallID, Timestamp, Usage.
- `Usage` — [`store.go:32-35`](../../sessions/store.go) — InputTokens, OutputTokens.
- `Store` — [`store.go:54-56`](../../sessions/store.go) — `dir string`.
- `NewStore(dir) *Store` — [`store.go:68`](../../sessions/store.go).
- `(*Store).CreateSession(agentName) Session` — [`store.go:73`](../../sessions/store.go) — UUID via `github.com/google/uuid`.
- `(*Store).AppendMessage(sessionID, Message) error` — [`store.go:91`](../../sessions/store.go) — Append-only JSONL line.
- `(*Store).LoadSession(sessionID) (Session, []Message, error)` — [`store.go:118`](../../sessions/store.go).
- `(*Store).ListSessions() ([]Session, error)` — [`store.go:151`](../../sessions/store.go) — Scans dir for `.jsonl`.
- `(*Store).DeleteSession(sessionID) error` — [`store.go:171`](../../sessions/store.go).

## Pluggability hooks

A lone compile-time interface assertion ([`store.go:58-66`](../../sessions/store.go)) — but no other type implements it, no package imports it. As far as depth-1 goes, this is the **only** session store implementation, and it is parallel to (not part of) the `harness.SessionStore` lineage.

## External deps used

- `github.com/google/uuid` — session ID generation.
- stdlib: `bufio`, `encoding/json`, `fmt`, `os`, `path/filepath`, `strings`, `time`.

## Notable tests

- `TestCreateAndLoadSession` — [`store_test.go:7`](../../sessions/store_test.go) — Pins UUID + agentName + message ordering.
- `TestListSessions` — [`store_test.go:42`](../../sessions/store_test.go).
- `TestDeleteSession` — [`store_test.go:58`](../../sessions/store_test.go).

All use `t.TempDir()`.

## Open questions / gaps

1. **Why does this exist?** No caller. Either extracted-but-not-yet-wired, legacy from ghostfin, or earmarked for a CLI/local tool.
2. **No overlap with `harness/lite` — answered.** `harness/lite/` does not ship a `LiteSessionStore`; only seven of the eight Layer 1 interfaces are implemented. `sessions/` is therefore the only session-log impl in the module today, and it is structurally unrelated to `harness.SessionStore`.
3. **No tenancy or context.** Cannot satisfy `harness.SessionStore`; would need a wrapper if it ever gets adapted.
4. **No fsync.** `AppendMessage` writes JSONL but doesn't fsync — fine for desktop, risky if reused under crash conditions.
5. **No purpose statement.** No package doc-comment, no README; reviewer should ask the maintainer where this is meant to live.
