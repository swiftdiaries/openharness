# Architecture

openharness is organized into six layers. Each layer is a standalone PR chain with its own tests; lower layers don't import higher ones.

## The layers

1. **Layer 1 — Harness boundary.** Interfaces and Lite implementations for sessions, agents, tools, secrets, artifacts, runners, skills, and channels. The boundary is the seam between the framework and any backend (local files, Postgres, S3, Vault, K8s).
2. **Layer 2 — Agent primitives.** Agent loop (think → act → observe), tool system with per-definition effects and deny patterns, LLM provider abstraction (Anthropic, OpenAI-compatible, OpenRouter).
3. **Layer 3 — Infrastructure.** Cost tracking, two-layer tracing (relational metadata + columnar span data), session management, config, edition gating, scheduler.
4. **Layer 4 — App scaffold.** Knowledge graph, HTTP gateway + SSE, agent persona system, and a Wails app lifecycle for desktop deployments.
5. **Layer 5 — Enterprise implementations.** Postgres / S3 / Vault / K8s / Lambda backends for the Layer 1 interfaces.
6. **Layer 6 — End-user SDK.** Versioned HTTP+JSON+SSE contract with typed Go clients. Two transports share one API: an in-process transport for Lite and an HTTP+SSE client for Enterprise.

**Lite vs Enterprise.** Lite is in-process and zero-network — the same binary runs the agent loop, stores sessions on the filesystem, and talks to the UI over local channels. Enterprise is a server: sessions live in Postgres, artifacts in S3, secrets in Vault, runners dispatch to K8s or Lambda, spans export via OTLP. Both sit behind the same Layer 1 interfaces.

## Import rules

- openharness imports nothing from any vertical.
- Verticals import openharness; they add domain tools, agents, and UI.
- Vertical-specific enterprise repos may import both openharness and their own vertical.

## Customization

Verticals customize at app startup via registration APIs — register domain tools, seed agent definitions, add trace attributes, mount frontend views. No framework forking.

## Deeper reading

- Full extraction design: [2026-04-10 openharness extraction](superpowers/specs/2026-04-10-openharness-extraction-design.md)
- Layer 2 (agent primitives): [2026-04-13 Layer 2 design](superpowers/specs/2026-04-13-openharness-layer-2-agent-primitives-design.md)
- Layer 6 (end-user SDK): [2026-04-17 Layer 6 SDK](superpowers/specs/2026-04-17-layer-6-sdk-design.md)
