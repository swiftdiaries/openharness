# openharness

[![CI](https://github.com/swiftdiaries/openharness/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/swiftdiaries/openharness/actions/workflows/ci.yml)

**A Go framework for building agentic AI platforms.**

> **Status: alpha.**
> v0.1.0 ships the harness boundary.
> Agent primitives are landing in Layer 2.
> See the [roadmap](docs/roadmap.md).

openharness handles the plumbing for agent apps: a harness boundary, an agent loop, a tool system, a provider abstraction, cost tracking, and tracing.
You extend it by registering tools, agents, and views at startup.

The same codebase runs two ways.
**Lite** runs locally with zero network.
**Enterprise** runs in the cloud with multi-tenant, durable state.
To build a new vertical, import the framework and wire in your domain.

## What ships today

- **Harness boundary.** Interfaces and Lite implementations for sessions, agents, tools, secrets, artifacts, runners, skills, and channels.
- **Agent primitives.** Agent loop, tool system, and provider abstraction (Anthropic, OpenAI-compatible, OpenRouter). Landing in Layer 2.

## What's coming

- App scaffold (Wails)
- Enterprise backends (Postgres, S3, Vault, Kubernetes)
- End-user SDK

See [docs/roadmap.md](docs/roadmap.md) for the full plan.

## Using openharness

Requires Go 1.25+.
The module is `github.com/swiftdiaries/openharness`.

There's no v1 SDK yet.
The planned `openharness.NewApp(...)` one-call entry point is on the [roadmap](docs/roadmap.md) and isn't shipped.
Today you consume openharness as a set of primitives.

## Learn More

- **[Architecture](docs/architecture.md):** what's available today and how the pieces fit together
- **[Roadmap](docs/roadmap.md):** layers, milestones, and what's next
- **[Contributing](docs/contributing.md):** dev setup and workflow
- **[beads](.beads):** where issue tracking lives

## Contributing

- Work is tracked using [beads](https://github.com/steveyegge/beads)
- Design specs and execution plans live under `docs/superpowers/{specs,plans}/`

## License

TBD.
