# Layer 6 — End-User SDK Design

**Depends on:** Layer 2 Plans 1–6 (agent loop, providers, tools, MCP, app registration) complete.

## Context

Layers 1–5 of openharness produce a Go framework for building agent verticals in-process. Layer 6 is the end-user SDK: how developers *outside* the framework consume openharness. The references the user pointed at are OpenAI's Agents SDK (in-process, per-language) and Anthropic's Managed Agents SDK (typed client for a hosted service). Layer 6 picks the hosted-service shape because it matches the Enterprise deployment model and keeps the language SDK story simple: one API contract, N clients generated from it.

Goal: let a developer create an agent, open a session, stream events, handle tool calls, and attach skills — via a typed client — against either local Lite or remote Enterprise, without changing the client API.

## Non-goals (v1)

TS/Python SDKs (codegen-ready but not built), Environments resource, Vaults/Credentials resource, Files/Artifacts as first-class SDK surface, OIDC/SSO, mTLS, per-tenant k8s namespaces, cross-region replication, user-authored built-in tools, gRPC.

## Architecture

One API contract, two transports, three tool types, two deployment tiers.

- **API contract** — HTTP + JSON + SSE, versioned under `/v1`. Agent-centric resources: `Agents`, `AgentVersions`, `Sessions`, `Skills`. Session events are an SSE stream; session input is `POST /sessions/{id}/messages` (user messages and custom-tool results).
- **Lite** — openharness Go framework runs in-process behind `harness.HarnessRunner`. No network, no DBOS, no durability; documented as the playground tier.
- **Enterprise** — same Go core deployed as a k8s service, multi-tenant, DBOS-backed durable execution, workspace-scoped API keys via `Authorization: Bearer`.
- **Go SDK** — module `github.com/swiftdiaries/openharness-sdk-go`, one surface, two transports: `InProcess(harness.HarnessRunner)` and `HTTP(baseURL, apiKey)`. Transport chosen at construction; client API identical.
- **Tool types (`tool.type` discriminator):**
  - `builtin` — runs server-side (9 built-ins from Layer 2 Plan 3).
  - `mcp` — remote MCP URL server, registered per-agent.
  - `custom` — schema declared on the agent; invocation streams a `tool_use` event to the client, which executes locally and posts a `tool_result` back. Matches Anthropic Managed Agents' custom tools.
- **Non-Go clients** — deferred. SDK is defined spec-first (OpenAPI + SSE event schema) so TS/Python clients can be generated on demand.

## Resource schema (v1)

All endpoints under `/v1`, all requests `Authorization: Bearer <api_key>`. Lite accepts any key.

### Agents
- `POST /agents` — `{name, description?, model, system?, tools[], skills[], metadata?}` → `{id, version_id, ...}`. Creates version 1.
- `GET /agents/{id}` · `GET /agents` (paginated) · `PATCH /agents/{id}` (creates new version) · `POST /agents/{id}/archive`.

### AgentVersions
Immutable snapshots for prod rollouts.
- `GET /agents/{id}/versions` · `GET /agents/{id}/versions/{version_id}`.
- `PATCH /agents/{id}` with `pin_version: "v3"` pins; omit to auto-advance.

### Sessions
One conversation thread against one agent version.
- `POST /sessions` — `{agent_id, agent_version?, initial_message?, metadata?}` → `{id, status}`.
  - Status values: `running | waiting_for_input | waiting_for_tool | completed | failed`.
- `GET /sessions/{id}` · `GET /sessions` (filter by `agent_id`, `status`).
- `POST /sessions/{id}/messages` — discriminated body: `{type: "user_message", content}` or `{type: "tool_result", tool_use_id, content, is_error?}`. Resumes a paused session.
- `GET /sessions/{id}/events` — SSE stream, see below.
- `POST /sessions/{id}/cancel` · `POST /sessions/{id}/interrupt`.

### Skills
- `POST /skills` (multipart bundle upload) · `GET /skills` · `GET /skills/{id}` · `POST /skills/{id}/archive`.
- Agents reference via `skills: [{skill_id, version}]`.

### Tool definition (inline on agent)

```json
{ "type": "builtin", "name": "web_search", "config": {} }
{ "type": "mcp",     "name": "my-mcp",    "url": "https://...", "headers": {} }
{ "type": "custom",  "name": "check_inventory", "description": "...", "input_schema": {} }
```

Server executes `builtin` and `mcp`; client executes `custom`.

## Session event stream (SSE)

`GET /v1/sessions/{id}/events` — `text/event-stream`. Each event has `event:` type and `data:` JSON. Resumable via `Last-Event-ID` header; server replays from the `SessionStore` event log starting after that ID.

Event types (v1):

- `session.status` — `{status, reason?}`. Emitted on every transition.
- `message.start | message.delta | message.stop` — assistant text streaming. Mirrors Anthropic streaming shape.
- `tool_use` — `{tool_use_id, tool_name, tool_type, input}`. For `builtin`/`mcp`: informational, a `tool_result` event follows. For `custom`: session → `waiting_for_tool`, client must post a `tool_result` via `/messages` to resume.
- `tool_result` — `{tool_use_id, content, is_error}`. Emitted after server-side execution, or echoed after the client posts one.
- `cost.update` — `{session_total_tokens, session_total_usd, last_call: {input, output, cache_read, cache_create}}`. Sourced from Layer 2 Plan 1 cost tracker.
- `ask_user` — `{prompt_id, question}`. Session → `waiting_for_input`; resume with `{type: "user_message", in_reply_to: prompt_id}`.
- `error` — `{code, message, retryable}`. Terminal unless retryable.
- `ping` — keepalive every 15s (LB idle-timeout defense).

**Split:** SSE is server→client only. All client→server input (user messages, tool results, cancel, interrupt) goes through POST endpoints. Matches what works cleanly behind AWS ALB/NLB.

## Durable execution (Enterprise) — DBOS

Each session = one DBOS workflow. Loop iterations, LLM calls, and tool invocations are DBOS steps. Steps are checkpointed to Postgres before execution, results persisted after. On pod crash, DBOS replays the workflow, skips completed steps, and resumes at the first unfinished one.

- **LLM call** → DBOS step. Response persisted; replay returns stored response (no re-billing).
- **Built-in tool call** → DBOS step. Idempotency key = `session_id:tool_use_id`. Non-idempotent built-ins (exec) commit intent → execute → commit result.
- **MCP tool call** → DBOS step. Idempotency key forwarded as `Idempotency-Key` header when supported.
- **Custom tool** → DBOS workflow yields via durable `recv()`; resumes on `POST /messages` with `tool_result`. Wait survives pod restarts.
- **Event emission** → written to `SessionStore` inside the same Postgres transaction as the DBOS step commit; SSE replay and workflow state cannot diverge.
- **Ownership/recovery** → DBOS recovery sweeper reclaims orphaned workflows on pod death. No bespoke lease logic.

Lite does not run DBOS. Crash = loss of in-flight step; documented as an intentional tier difference. API contract is identical across tiers; the only observable difference is crash-recovery behavior.

## Auth and multi-tenancy

- **Auth**: API keys scoped to a workspace. `Authorization: Bearer <key>`. Lite accepts any key (or a single shared dev key).
- **Tenancy**: single control-plane Postgres partitioned by `workspace_id` derived from the API key. Per-tenant k8s namespaces are a deployment option, not a v1 requirement.
- **OIDC/SSO** for the admin dashboard is a separate workstream, not part of the SDK spec.

## Go SDK surface

Module: `github.com/swiftdiaries/openharness-sdk-go`. Separate from the framework module. Depends on the framework only for InProcess transport type definitions.

```go
c, _ := openharness.NewClient(openharness.HTTP("https://...", "sk-..."))
c, _ := openharness.NewClient(openharness.InProcess(runner))  // harness.HarnessRunner

c.Agents.Create / Get / List / Update / Archive
c.Agents.Versions.List / Get
c.Sessions.Create / Get / List / SendMessage / Cancel / Interrupt
c.Skills.Upload / List / Get / Archive

stream, _ := c.Sessions.Events(ctx, id, EventsParams{LastEventID: "..."})
for stream.Next() { ev := stream.Event() /* typed */ }

// Convenience wrapper: drives Events() + auto-posts tool_result via SendMessage.
c.Sessions.Run(ctx, id, openharness.ToolHandlers{
    "check_inventory": func(ctx, input) (any, error) { ... },
})
```

**Transport interface:** small `Transport` abstraction (`Do(req) (resp, stream, err)`) with `HTTP` and `InProcess` implementations. HTTP uses `net/http` + `bufio`-based SSE parser. InProcess adapts `HarnessRunner.Dispatch` + framework event channel to the SDK's typed event stream.

## Non-Go clients — codegen plan (deferred)

- Spec artifacts committed in the openharness repo: `api/openapi.yaml` (control plane) and `api/events.md` + `api/events.schema.json` (SSE event schema).
- Generator of record: openapi-generator for REST; event schema codegen'd via `quicktype` or hand-rolled thin types per language.
- Reserved package names: `openharness` (PyPI), `@openharness/sdk` (npm).
- Trigger to build: (a) paying Enterprise tenant or external vertical asks for it, or (b) an internal non-Go repo needs a client. Until then, the spec artifacts are the only deliverable.

## Deployment (Enterprise)

- **Workloads:** stateless `openharness-api` deployment (HTTP+SSE), `openharness-worker` deployment (DBOS workflow executors). Same binary, different entrypoints.
- **Datastore:** Postgres (DBOS state + SessionStore events + agents/versions/skills metadata). Managed Postgres recommended.
- **Object store:** S3-compatible for skill bundles and artifacts.
- **Ingress:** AWS ALB (or any HTTP LB) with SSE idle timeout ≥ 300s; 15s `ping` keepalives.
- **Scaling:** API pods on request rate; worker pods on DBOS queue depth.
- **Helm chart:** `deploy/helm/openharness-enterprise` — separate deliverable, referenced not designed here.

## Milestones

- **v1 (Layer 6 release):** HTTP+SSE API, Go SDK (both transports), DBOS durable execution in Enterprise, Agents/Versions/Sessions/Skills, tools (builtin/mcp/custom), workspace-scoped Bearer auth, OpenAPI + event-schema artifacts.
- **Depends on:** Layer 2 Plans 1–6 complete.
- **Post-v1:** TS/Python SDKs via codegen (on demand), Environments, Vaults/Credentials, Files/Artifacts, OIDC/SSO, Helm chart hardening.

## Verification

End-to-end checks once implemented:

1. Create agent → create session → stream events → assistant streams text → session completes. Both transports.
2. Custom tool round-trip: server emits `tool_use` for a `type:"custom"` tool; client posts `tool_result`; session resumes and completes.
3. Durable execution: kill API pod mid-LLM-call; session resumes on another pod without re-billing and without data loss. Kill worker pod mid-tool-call; tool is not re-executed if idempotent result was committed.
4. SSE resume: drop the client connection mid-stream; reconnect with `Last-Event-ID`; missed events replayed.
5. Lite parity: same Go SDK code path works against `InProcess(runner)` and against a local `openharness serve` — identical observable behavior except crash recovery.
6. Auth: requests without Bearer token rejected on Enterprise; cross-workspace data access impossible via API key scoping.
