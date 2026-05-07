# openspec

First-depth walkthrough of `openspec/`.

## What lives here

`openspec/` is an OpenSpec workspace — a spec-driven workflow for proposing, designing, and tracking changes. It is metadata, not code. The repo uses it alongside beads (issue tracker) and `docs/superpowers/` (design specs / execution plans).

## Top-level files

- `openspec/config.yaml` — Minimal: `schema: spec-driven`. Project name, tech-stack context, and per-artifact rules are all unset; the file is mostly placeholder.

## Subdirectories

- `openspec/changes/` — Two entries:
  - `archive/` — Empty placeholder for completed changes.
  - `platform-builder-runtime/` — Active change. Contains `proposal.md`, a 17 KB `design.md`, a 13 KB `tasks.md`, and a `specs/` subdirectory.
- `openspec/specs/` — Empty. No frozen specifications yet.

## Active vs archived changes

**Active**

- `platform-builder-runtime` (created 2026-04-25). Proposes a runtime for composing agentic applications as "a machine for building machines." Frames Layers 2–6 around three primitives: a composition entrypoint (`compose.New(profile, opts...)`), named profiles as first-class artifacts (`profiles.Lite()`, custom profiles), and declarative composition via `openharness.yaml`.

**Archived**

- None yet.

## Open questions / gaps

1. **`config.yaml` is unfilled** — set project name and tech-stack context before more changes accumulate.
2. **`specs/` is empty** — no frozen spec has been promoted from a completed change yet.
3. **Single in-flight change** — `platform-builder-runtime` is the only one. Reviewer should read its `proposal.md` and `design.md` to see whether the planned `compose.New` / profile shape is consistent with what the code already exposes (especially `harness/` and `harness/lite/`).
4. **Relationship to `docs/superpowers/`** — both directories hold design intent. Reviewer should ask whether OpenSpec is the new canonical location and `superpowers/` is being migrated, or whether they coexist.
