# cost

First-depth walkthrough of `cost/`.

## Package summary

`cost` provides per-token pricing lookup and CSV-backed spend tracking with daily/monthly budget gates. Pricing has a built-in catalog and a global override map; tracking is mutex-protected file I/O.

**Importers (today):** zero. The package is freestanding utility code awaiting integration into the agent loop / harness.

## Files at depth 1

- `cost/pricing.go` (118 lines) — Catalog, override hook, lookup, calculation, format.
- `cost/tracker.go` (136 lines) — CSV append-only spend log + budget check.
- `cost/pricing_test.go` (78 lines) — Lookup, fallback, free models, override.
- `cost/tracker_test.go` (68 lines) — Record + CheckBudget.

## Public surface

### Types

- `ModelPrice` — [`pricing.go:9-12`](../../cost/pricing.go) — `InputPerToken`, `OutputPerToken`.
- `Record` — [`tracker.go:13-21`](../../cost/tracker.go) — Timestamp, Provider, Model, token counts, CostUSD, SessionID.
- `Tracker` — [`tracker.go:23-28`](../../cost/tracker.go) — File path, daily/monthly limits, mutex.

### Exported functions

- `SetPricingOverrides(map[string]ModelPrice)` — [`pricing.go:67`](../../cost/pricing.go).
- `Calculate(model, inputTokens, outputTokens) float64` — [`pricing.go:103`](../../cost/pricing.go).
- `FormatCost(float64) string` — [`pricing.go:109`](../../cost/pricing.go).
- `NewTracker(dataDir, dailyLimit, monthlyLimit) (*Tracker, error)` — [`tracker.go:30`](../../cost/tracker.go).
- `(*Tracker).Record(Record) error` — [`tracker.go:41`](../../cost/tracker.go).
- `(*Tracker).CheckBudget() (bool, string)` — [`tracker.go:65`](../../cost/tracker.go).

### Package-level state

- `defaultPricing map[string]ModelPrice` — [`pricing.go:14-63`](../../cost/pricing.go) — 13 entries; immutable after init.
- `pricingOverrides map[string]ModelPrice` — [`pricing.go:65`](../../cost/pricing.go) — **Unprotected mutable global.**

## Pluggability hooks

`SetPricingOverrides` is the single override seam ([`pricing.go:67`](../../cost/pricing.go)). Lookup order ([`pricing.go:71-100`](../../cost/pricing.go)):

1. Strip leading `openrouter/` prefix.
2. Exact match in overrides.
3. Exact match in defaults.
4. Substring match in overrides, then defaults.
5. Fallback to `anthropic/claude-sonnet-4-6` pricing.

Verticals can register custom models without modifying the catalog. The fallback to Sonnet means unknown models never error — good for resilience, bad for visibility.

## Concurrency / state model

- `Tracker.mu` ([`tracker.go:27`](../../cost/tracker.go)) protects file I/O on `Record` and `CheckBudget`.
- **`pricingOverrides` is unsynchronized** ([`pricing.go:65`](../../cost/pricing.go)). `SetPricingOverrides` writes to a package-level map while concurrent `Calculate` calls read it. Tests use `defer SetPricingOverrides(nil)` ([`pricing_test.go:70`](../../cost/pricing_test.go)) — fine for test cleanup, racy in production. Likely tolerable since overrides are typically configured once at startup, but should be set-once or guarded by a mutex.

## External deps used

stdlib only: `encoding/csv`, `fmt`, `os`, `path/filepath`, `strconv`, `strings`, `sync`, `time`.

## Notable tests

- `TestCalculateKnownModel` — [`pricing_test.go:5`](../../cost/pricing_test.go) — Pins Sonnet $3 in / $15 out per million.
- `TestCalculateSubstringMatch` — [`pricing_test.go:13`](../../cost/pricing_test.go) — `claude-3-5-sonnet-20241022` matches the bare `claude-3-5-sonnet` entry.
- `TestCalculateUnknownFallsBackToSonnet` — [`pricing_test.go:21`](../../cost/pricing_test.go) — Documents fallback behavior.
- `TestCalculateFreeModels` — [`pricing_test.go:47`](../../cost/pricing_test.go) — `:free` suffix → 0.
- `TestCalculateWithConfigOverride` — [`pricing_test.go:66`](../../cost/pricing_test.go) — Pluggability seam.
- `TestRecordAndCheckBudget` — [`tracker_test.go:9`](../../cost/tracker_test.go) — Daily limit enforcement.

## Open questions / gaps

1. **Race on `pricingOverrides`.** Either guard with `sync.RWMutex` or document set-once semantics.
2. **Silent fallback to Sonnet.** `Calculate("typo-model", ...)` quietly returns Sonnet pricing — no log, no metric, no warning. For a cost-tracking system this hides accuracy bugs.
3. **CSV is the only format.** No DB pluggability; migration would require a new tracker.
4. **`CheckBudget` is read-after-write racy.** Two goroutines can both observe "under budget" then both `Record` over the limit. Acceptable for soft enforcement but worth documenting.
5. **Substring match collisions.** `"sonnet"` substring in any overlay key matches multiple Anthropic Sonnet entries; ordering is map-iteration order, i.e. nondeterministic.
6. **No integration with the agent loop.** Records have to be written explicitly by callers; nothing today calls `tracker.Record(...)`.
