# Contributing

## Dev setup

- Go 1.25+. Module path: `github.com/swiftdiaries/openharness`.
- `go test ./...` runs the full test suite.
- Scope to a single package with e.g. `go test ./tools/core/...`.

## Change planning and task tracking

This repo uses OpenSpec as the source of truth for planned changes and implementation progress:

- `openspec list --json` — list active changes
- `openspec status --change <name>` — inspect artifact and task status
- `openspec validate <name>` — validate an affected change
- `openspec/changes/<name>/tasks.md` — update implementation progress

Use `/opsx:propose`, `/opsx:apply`, and `/opsx:archive` for the change lifecycle. See `AGENTS.md` for the full agent workflow and session-completion protocol.

## Design docs and plans

- Design specs live under `docs/superpowers/specs/` (one per layer or major sub-design).
- Execution plans live under `docs/superpowers/plans/` (one per PR chain).
- Each layer ships as an independent PR chain with tests passing before merge.

## Agent-specific instructions

If you're working on this repo with Claude Code or another agent, see `CLAUDE.md` and `AGENTS.md` for the expected workflow.
