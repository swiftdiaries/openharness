# Project Instructions for AI Agents

This project uses **OpenSpec** for change planning and task tracking.

## Workflow

- Inspect active work with `openspec list --json`.
- Treat `openspec/changes/<name>/tasks.md` as the source of truth for implementation status.
- Use `/opsx:propose`, `/opsx:apply`, and `/opsx:archive` for the change lifecycle.
- Validate affected changes with `openspec validate <name>` before completion.
- Run the relevant tests and linters, then commit and push all intended changes.

## Build and test

```bash
go test ./...
go vet ./...
```

## Architecture overview

See `docs/architecture.md` for the layered framework design and `docs/roadmap.md` for delivery status.

## Conventions

- Design specs and execution plans live under `docs/superpowers/{specs,plans}/`.
- OpenSpec changes live under `openspec/changes/` and carry proposal, design, capability specs, and tasks.
- Prefer non-interactive shell commands and preserve unrelated working-tree changes.
