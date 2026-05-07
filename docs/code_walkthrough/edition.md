# edition

First-depth walkthrough of `edition/`.

## Package summary

`edition` is a tiny capability-flag registry that distinguishes a Lite vs. Enterprise build at runtime via an `atomic.Pointer[Edition]` global. It defines two factory constructors and a small set of feature flags (RBAC, Sync, Admin, Billing, Scheduler, DBBackend) plus agent/team caps.

**Importers (today):** zero in the rest of the repo. Package is defined but not yet wired to gate any feature. Tests are self-contained.

## Files at depth 1

- `edition/edition.go` (56 lines).
- `edition/edition_test.go` (98 lines).

## Public surface

### Types

- `Edition` — [`edition.go:5-15`](../../edition/edition.go) — Name, MaxAgents, MaxTeams, HasRBAC, HasSync, HasAdmin, HasBilling, HasScheduler, DBBackend.

### Functions

- `Lite() Edition` — [`edition.go:31`](../../edition/edition.go) — `MaxAgents: 5, MaxTeams: 1`, all flags false, sqlite.
- `Enterprise() Edition` — [`edition.go:39`](../../edition/edition.go) — `MaxAgents: 0` (unlimited), all flags true, postgresql.
- `(Edition).IsLimited() bool` — [`edition.go:18`](../../edition/edition.go).
- `(Edition).AgentLimitReached(current int) bool` — [`edition.go:24`](../../edition/edition.go).
- `Current() Edition` — [`edition.go:54`](../../edition/edition.go) — Atomic load.
- `SetEdition(Edition)` — [`edition.go:55`](../../edition/edition.go) — Atomic store.

### Package-level state

- `current atomic.Pointer[Edition]` — [`edition.go:47`](../../edition/edition.go) — Initialized to `Lite()` in `init()` at [`edition.go:49-52`](../../edition/edition.go).

## Edition model

| Field | Lite | Enterprise |
|---|---|---|
| Name | "Lite" | "Enterprise" |
| MaxAgents | 5 | 0 (unlimited) |
| MaxTeams | 1 | 0 (unlimited) |
| HasRBAC | false | true |
| HasSync | false | true |
| HasAdmin | false | true |
| HasBilling | false | true |
| HasScheduler | false | true |
| DBBackend | sqlite | postgresql |

`Edition` is a value type. `Current()` returns a copy; goroutines that cache it won't see later mutations — usually desirable.

## Pluggability hooks

- **Custom edition?** Yes, structurally: `edition.Edition{...}` is a public struct with exported fields, and `SetEdition()` accepts any value. No factory registry, validation, or example. Intended use is the two factories; third editions are a back-door.
- **Independent flags?** Yes — set struct fields and call `SetEdition`.
- **No validation:** `SetEdition(Edition{MaxAgents: -1})` is accepted silently.

## External deps used

stdlib only: `sync/atomic`.

## Notable tests

- `TestEditionDefaults` — [`edition_test.go:5`](../../edition/edition_test.go) — Pins Lite as the package-init default.
- `TestEditionEnterprise` — [`edition_test.go:34`](../../edition/edition_test.go) — Pins Enterprise field values.
- `TestEditionConcurrency` — [`edition_test.go:63`](../../edition/edition_test.go) — 1000 concurrent Set/Read pairs; race-free under `sync/atomic.Pointer`.
- `TestLiteIsLimited` / `TestEnterpriseIsNotLimited` — [`edition_test.go:79-97`](../../edition/edition_test.go).

## Open questions / gaps

1. **Unused.** No call site in the codebase references `edition.Current()` or capability fields. Either gate features against it (e.g. agent-spawn limit) or delete.
2. **`init()`-time default.** Lite is hardcoded; no env var or config-driven selection.
3. **`MaxTeams` is set but never checked.** `AgentLimitReached` exists; no `TeamLimitReached`.
4. **Memory note.** Reviewers caching `Current()` in long-lived goroutines won't see runtime edition switches — fine, but worth documenting.
5. **Reviewer follow-up:** confirm the gateway / harness will wire `edition.Current().HasRBAC` (etc.) into actual gates.
