# docs

First-depth walkthrough of `docs/`.

## What lives here

The `docs/` directory holds the framework's narrative documentation: a layered architecture description, a release roadmap, contributor instructions, design specs and execution plans (under `superpowers/`), and this very `code_walkthrough/` directory.

This is documentation, not code. The walkthroughs in this folder cite `docs/architecture.md` and `docs/roadmap.md` as the framing the code is supposed to match — that comparison is the whole point of the review.

## Files at depth 1

- `docs/architecture.md` ([file](../architecture.md)) — The layered architecture: boundary → primitives → infrastructure → app scaffold → enterprise → SDK. Defines the Lite vs Enterprise split (in-process / zero network vs. server with Postgres + S3 + Vault + K8s/Lambda + OTLP). States that customization happens by **registering** tools, agents, and views at startup, not by forking.
- `docs/roadmap.md` ([file](../roadmap.md)) — Layer-by-layer release status. v0.1.0 shipped Layer 1 (boundary). Layer 2 (agent primitives) in progress; v0.2.0 targets Layer 2 complete plus MCP outbound bridge and UI bridge. Tracking epic `openharness-pol` in beads.
- `docs/contributing.md` ([file](../contributing.md)) — Dev setup (Go 1.25+, `go test ./...`), beads workflow, design-spec/execution-plan locations.

## Subdirectories at depth 1

- `docs/code_walkthrough/` — This directory. Per-package depth-1 walkthroughs plus the top-level [first_layer.md](./first_layer.md).
- `docs/superpowers/` — Houses `plans/` and `specs/`. Design specs and execution plans for each layer / PR chain.

## Key claims from architecture.md and roadmap.md

These are the assertions the code is being evaluated against:

- **Layered design.** Each layer is a standalone PR chain with its own tests; lower layers never import higher ones. (architecture.md)
- **Layer 1 = harness boundary.** Interfaces and Lite implementations for sessions, agents, tools, secrets, artifacts, runners, skills, channels. (architecture.md)
- **Layer 2 = agent primitives.** Agent loop (think → act → observe), tool system with effects + deny patterns, LLM provider abstraction (Anthropic, OpenAI-compat, OpenRouter). (architecture.md)
- **Lite vs Enterprise split** at the same seam — same binary semantics, different backends. (architecture.md)
- **Customization via registration, not forking.** (architecture.md)
- **v0.1.0 shipped Layer 1.** (roadmap.md)
- **v0.2.0 targets Layer 2 complete + MCP outbound + UI bridge.** (roadmap.md)
- **Layer-status tracking** lives in beads epic `openharness-pol`. (roadmap.md)

## Open questions / gaps

1. **Layer count discrepancy.** The README says "5-layer extraction"; architecture.md describes six layers. Reconcile.
2. **Design specs referenced under `superpowers/specs/`** — verify they exist and stay in sync with the public-facing docs. Roadmap acknowledges the table is hand-maintained.
3. **No per-layer doc template** in `contributing.md` — each PR chain decides its own doc shape.
4. **`AGENTS.md` and `CLAUDE.md` at the repo root** are referenced by `contributing.md` for the bd workflow but the contents are not summarized; reviewer should skim those once.
