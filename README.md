# openharness

A Go framework for building an agentic AI platform.

> **Status: alpha** — v0.1.0 shipped (harness boundary). Active work on agent primitives. See [roadmap](docs/roadmap.md).

openharness gives you the plumbing for agent-driven applications — a harness boundary, an agent loop, a tool system, a provider abstraction, cost tracking, and tracing — and lets you extend it by registering tools, agents, and views at startup. It targets two deployment shapes from one codebase: **Lite** (local/desktop, zero network) and **Enterprise** (cloud, multi-tenant, durable). Build a new vertical by importing the framework and plugging in your domain — no forking required.

## What's here today

- **Harness boundary** — interfaces and Lite implementations for sessions, agents, tools, secrets, artifacts, runners, skills, and channels.
- **Agent primitives** — agent loop, tool system, and provider abstraction (Anthropic, OpenAI-compatible, OpenRouter) — merging in Layer 2.
- **Planned, not yet here** — app scaffold (Wails), enterprise backends (Postgres / S3 / Vault / K8s), end-user SDK.
- → [docs/roadmap.md](docs/roadmap.md)

## Using openharness

Import path: `github.com/swiftdiaries/openharness`. Requires Go 1.25+.

The v1 end-user SDK — a one-call `openharness.NewApp(...)` entry point — is tracked as Layer 6 and is not yet shipped. Today the framework is consumable as primitives. See [docs/architecture.md](docs/architecture.md) for the shape and [docs/roadmap.md](docs/roadmap.md) for what's coming.

## Contributing

Work is tracked in [beads](https://github.com/steveyegge/beads). Design specs and execution plans live under `docs/superpowers/{specs,plans}/`. See [docs/contributing.md](docs/contributing.md) for dev setup and workflow.

## License

TBD.
