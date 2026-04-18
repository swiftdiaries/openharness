# openharness

**A Go framework for building agentic AI platforms.**

> **Status: alpha.** v0.1.0 ships the harness boundary. Agent primitives are landing in Layer 2. See the [roadmap](docs/roadmap.md).

openharness handles the plumbing for agent apps: a harness boundary, an agent loop, a tool system, a provider abstraction, cost tracking, and tracing. You extend it by registering tools, agents, and views at startup.

The same codebase runs two ways. **Lite** runs locally with zero network. **Enterprise** runs in the cloud with multi-tenant, durable state. To build a new vertical, import the framework and wire in your domain.

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

```go
import "github.com/swiftdiaries/openharness"
```

The v1 SDK (a one-call `openharness.NewApp(...)`) is Layer 6 and isn't shipped yet. For now, openharness is consumable as primitives. Read [docs/architecture.md](docs/architecture.md) for the shape and [docs/roadmap.md](docs/roadmap.md) for what's next.

## Contributing

Work is tracked in [beads](https://github.com/steveyegge/beads). Design specs and execution plans live under `docs/superpowers/{specs,plans}/`. See [docs/contributing.md](docs/contributing.md) for dev setup.

## License

TBD.
