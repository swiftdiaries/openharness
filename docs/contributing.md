# Contributing

## Dev setup

- Go 1.25+. Module path: `github.com/swiftdiaries/openharness`.
- `go test ./...` runs the full test suite.
- Scope to a single package with e.g. `go test ./tools/core/...`.

## Issue tracking: beads

This repo uses [beads](https://github.com/steveyegge/beads) for all task tracking. Quick reference:

- `bd ready` — next unblocked work
- `bd show <id>` — detailed view of an issue
- `bd update <id> --claim` — claim work atomically
- `bd close <id>` — mark complete

The tracking epic for the framework build-out is `openharness-pol`; the six layers are children (`openharness-pol.1` … `openharness-pol.6`), with Layer 1.5 — adoption enrichments for v0.2.0 — tracked alongside as `openharness-pol.7`. Layer 2 plans are grandchildren (`openharness-pol.2.1` … `openharness-pol.2.8`); use `bd graph openharness-pol.2` for the full Layer 2 chain. See `AGENTS.md` for the full workflow and the session-close protocol (commit, `bd dolt push`, `git push`).

## Design docs and plans

- Design specs live under `docs/superpowers/specs/` (one per layer or major sub-design).
- Execution plans live under `docs/superpowers/plans/` (one per PR chain).
- Each layer ships as an independent PR chain with tests passing before merge.

## Agent-specific instructions

If you're working on this repo with Claude Code or another agent, see `CLAUDE.md` and `AGENTS.md` for the expected workflow (non-interactive shell flags, session close, etc.).
