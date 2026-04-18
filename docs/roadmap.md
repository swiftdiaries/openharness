# Roadmap

openharness ships in six layers. Each layer is an independent PR chain designed to be reviewable and testable on its own.

| Layer | Status | Spec |
|-------|--------|------|
| Layer 1 — harness boundary | shipped (v0.1.0) | [extraction design](superpowers/specs/2026-04-10-openharness-extraction-design.md) |
| Layer 2 — agent primitives | in progress (Plans 1–3 merged; Plan 4 next) | [Layer 2 design](superpowers/specs/2026-04-13-openharness-layer-2-agent-primitives-design.md) |
| Layer 3 — infrastructure | not started | — |
| Layer 4 — app scaffold | not started | — |
| Layer 5 — enterprise implementations | not started | — |
| Layer 6 — end-user SDK | design complete | [Layer 6 SDK design](superpowers/specs/2026-04-17-layer-6-sdk-design.md) |

**Next release.** v0.2.0 ships with Layer 2 complete — agent loop, tool system, and provider abstraction — plus the MCP outbound bridge and UI bridge from Plan 4.

**Live status.** For up-to-the-minute state, run `bd ready` in a checkout, or inspect the tracking epic `openharness-pol` (`bd graph openharness-pol.6` for the full layer chain).

> This table is maintained by hand. The authoritative source is the `## Progress` section of [the extraction design](superpowers/specs/2026-04-10-openharness-extraction-design.md); update both when layer status changes.
