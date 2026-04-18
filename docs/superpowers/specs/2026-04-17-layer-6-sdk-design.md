# Layer 6 — End-User SDK Design

**Depends on:** Layers 1–5 for the full Enterprise release. A Lite-only in-process-transport slice can ship earlier — it needs Layer 2 (agent primitives, providers, tools, MCP, app registration) plus the `sessions` and `config` bits of Layer 3. The HTTP server lives in Layer 4 (`gateway`); DBOS durable execution, multi-tenant Postgres, and Vault-backed Secrets come from Layer 5.

## Context

Layers 1–5 of openharness produce a Go framework for building agent verticals in-process. Layer 6 is the end-user SDK: how developers *outside* the framework consume openharness. The references the user pointed at are OpenAI's Agents SDK (in-process, per-language) and Anthropic's Managed Agents SDK (typed client for a hosted service). Layer 6 picks the hosted-service shape because it matches the Enterprise deployment model and keeps the language SDK story simple: one API contract, N clients generated from it.

Goal: let a developer create an agent, open a session, stream events, handle tool calls, attach skills, and manage credentials — via a typed client — against either local Lite or remote Enterprise, without changing the client API.

## Non-goals (v1)

TS/Python SDKs (codegen-ready but not built), Environments resource, Files/Artifacts as first-class SDK surface, OIDC/SSO, mTLS, per-tenant k8s namespaces, cross-region replication, user-authored built-in tools, gRPC, code-executing skill bundles (declarative-only in v1; see Sandboxes).

## Architecture

One API contract, two transports, three tool types, two deployment tiers, one sandbox abstraction.

- **API contract** — HTTP + JSON + SSE, versioned under `/v1`. Agent-centric resources: `Agents`, `AgentVersions`, `Sessions`, `Skills`, `Secrets`, `Webhooks`, `AuditEvents`. Session events are an SSE stream; session input is `POST /sessions/{id}/messages`.
- **Lite** — openharness Go framework runs in-process behind `harness.HarnessRunner`. No network, no DBOS, no durability; documented as the playground tier. **Refuses to bind non-loopback addresses unless `OPENHARNESS_DEV_KEY` is set; once set, Bearer auth is enforced.** Loopback bind accepts any key for true zero-config dev.
- **Enterprise** — same Go core deployed as a k8s service, multi-tenant, DBOS-backed durable execution, workspace-scoped API keys via `Authorization: Bearer`.
- **Go SDK** — three modules: `openharness-sdk-go` (types + transport interface), `openharness-sdk-go/httpclient`, `openharness-sdk-go/inprocess`. HTTP-only consumers don't pull framework deps.
- **Tool types (`tool.type` discriminator):**
  - `builtin` — runs server-side (built-ins from Layer 2 Plan 3). `exec` is gated on a (sandbox, trust_mode) admission rule (see Sandboxes).
  - `mcp` — remote MCP URL server, registered per-agent. Headers reference Secrets, never plaintext on non-loopback (see Secrets).
  - `custom` — schema declared on the agent; invocation streams a `tool_use` event to the client, which executes locally and posts a `tool_result` back. Server validates inputs against `input_schema` (JSON Schema 2020-12) before emit and on `tool_result` ingest.
- **Sandboxes** — first-class config naming *where untrusted code runs*. Required for the `exec` built-in and reserved for code-executing skill bundles post-v1.
- **Secrets** — minimal KEK-wrapped resource for MCP headers and BYO model-provider keys.
- **Non-Go clients** — deferred. SDK is defined spec-first (OpenAPI + SSE event schema) so TS/Python clients can be generated on demand.

## Resource schema (v1)

All endpoints under `/v1`, all requests `Authorization: Bearer <api_key>`. Lite loopback bind accepts any key; non-loopback bind requires `OPENHARNESS_DEV_KEY`.

### Agents
- `POST /agents` — `{name, description?, model, system?, tools[], skills[], sandbox?, trust_mode?, model_provider_key?, metadata?}` → `{id, version_id, ...}`. Creates version 1.
- `GET /agents/{id}` · `GET /agents` (paginated) · `PATCH /agents/{id}` (creates new version) · `POST /agents/{id}/archive`.

### AgentVersions
Immutable snapshots for prod rollouts.
- `GET /agents/{id}/versions` · `GET /agents/{id}/versions/{version_id}`.
- Sessions snapshot the agent's `latest_version` at session-create time. To opt into auto-advance, set `auto_advance: true` on session create; otherwise the version is fixed for the life of the session.

### Sessions
One conversation thread against one agent version.
- `POST /sessions` — `{agent_id, agent_version?, auto_advance?, initial_message?, max_tokens?, max_usd?, metadata?}` → `{id, status}`.
  - Status values: `running | waiting_for_input | waiting_for_tool | completed | failed | cancelled`.
  - Requires `Idempotency-Key` header (24h dedup window).
  - `?stream=true` query param: same response upgrades to SSE event stream, eliminating the create-then-subscribe race.
- `GET /sessions/{id}` · `GET /sessions` (filter by `agent_id`, `status`).
- `POST /sessions/{id}/messages` — discriminated body: `{type: "user_message", content, in_reply_to?}` or `{type: "tool_result", tool_use_id, content, is_error?}`. Resumes a paused session. Requires `Idempotency-Key`.
- `GET /sessions/{id}/events` — SSE stream, see below.
- `POST /sessions/{id}/cancel` — terminal; session moves to `cancelled`, no further input accepted.
- `POST /sessions/{id}/interrupt` — stops the current step (LLM call or tool); session moves to `waiting_for_input` and is resumable via `/messages`.
- `DELETE /sessions/{id}` — soft-delete; PII-scrub job redacts message content after 30d (workspace-configurable). Metrics rows preserved.

### Skills
- `POST /skills` (multipart bundle upload) · `GET /skills` · `GET /skills/{id}` · `POST /skills/{id}/archive`.
- Agents reference via `skills: [{skill_id, version}]`.
- v1 bundles are declarative (instructions + assets); SHA-256 manifest required, optional Cosign signature, 50MB size cap. Code-executing bundles are deferred to post-v1 and will inherit the Sandbox admission rule.

### Secrets
Minimal credential store for MCP headers and BYO model-provider keys.
- `POST /secrets` — `{name, value}` → `{id, name, created_at}`. Value never returned after creation.
- `GET /secrets` (lists names + metadata only) · `DELETE /secrets/{name}`.
- Storage: per-workspace DEK encrypts the value; DEK is wrapped by a KEK from AWS KMS / GCP KMS / HashiCorp Vault (deployment-configured). Plaintext never touches Postgres.
- Reference syntax: `${secret:name}` in MCP `headers` values and `model_provider_key` field. Server resolves at call time.

### Webhooks
Async completion callbacks for integrators that don't want long-lived SSE.
- `POST /webhooks` — `{url, events[], description?}` → `{id, secret, ...}`. Subscribable events: `session.completed`, `session.failed`, `session.cancelled`, `tool_use` (custom only), `cost.threshold_exceeded`.
- `GET /webhooks` · `DELETE /webhooks/{id}`.
- Delivery: HMAC-SHA256 signed via `X-Openharness-Signature: sha256=<hex>` using the per-webhook `secret`. Server retries with exponential backoff up to 24h.

### AuditEvents
Append-only log for compliance and incident response.
- `GET /v1/audit_events` — paginated, filterable by `actor`, `resource_type`, `resource_id`, `action`, time range.
- Records CRUD on Agents, AgentVersions, Sessions, Skills, Secrets, Webhooks, API keys, and session lifecycle transitions (created, completed, failed, cancelled).
- Each event: `{id, timestamp, workspace_id, actor: {type, id}, action, resource: {type, id}, metadata}`.

### Models
- `GET /v1/models` — workspace-scoped list of supported models. Each entry: `{id: "provider/model", display_name, requires_byo_key, capabilities: {streaming, tools, vision, ...}}`.

### Tool definition (inline on agent)

```json
{ "type": "builtin", "name": "web_search", "config": {} }
{ "type": "mcp",     "name": "my-mcp",    "url": "https://...", "headers": {"Authorization": "Bearer ${secret:my_mcp_token}"}, "timeout_seconds": 120 }
{ "type": "custom",  "name": "check_inventory", "description": "...", "input_schema": { /* JSON Schema 2020-12 */ }, "output_schema": { /* optional */ } }
```

Server executes `builtin` and `mcp`; client executes `custom`. Server validates `custom` tool inputs against `input_schema` before emitting `tool_use` and validates `tool_result` content against `output_schema` (when declared) on ingest. Schema violations emit an `error` event with `code: "tool_validation"`.

## Sandboxes

A `Sandbox` names *where untrusted code runs* — required for the `exec` built-in, reserved for code-executing skill bundles post-v1.

```json
{ "type": "runloop" | "firecracker" | "gvisor" | "lambda" | "local",
  "config": { /* provider-specific */ } }
```

`trust_mode` on agent: `"restricted"` (default) | `"safe"`.

**`exec` admission rule** — server-enforced at agent creation:
- `exec` is permitted only if **all** of:
  - `sandbox.type ∈ {runloop, firecracker, gvisor, lambda}`
  - `trust_mode == "safe"`
  - For `lambda`: the agent's ID is in the deployment's `lambda_exec_allowlist`. Operators declare which agents have validated Lambda invocation patterns; isolation primitives (`runloop`, `firecracker`, `gvisor`) are sufficient evidence on their own.
- If an agent declares `exec` and the rule fails, agent creation returns `400 invalid_tool_config` naming the failing condition. No silent stripping.

Workspace defaults for (sandbox, trust_mode) are inheritable; agent-level fields override.

## MCP egress controls

- All outbound MCP calls go through the deployment's egress proxy.
- Default-deny: RFC1918 (10/8, 172.16/12, 192.168/16), 169.254/16 (IMDS), ::1, 127/8.
- TLS verification required (no `insecure_skip_verify`).
- Response size cap: 10MB per call (workspace can lower).
- Per-call timeout: **120s default, max 600s** (configurable per tool, capped at 600s without a deployment config change). MCP tool calls span DB queries, model inference, and file processing — the 600s ceiling matches realistic long-running patterns.

## Session event stream (SSE)

`GET /v1/sessions/{id}/events` — `text/event-stream`. Each event has `event:` type, `id:` ULID, `data:` JSON. Resumable via `Last-Event-ID` header; server replays from `SessionStore` after that ID. **Every reconnect re-validates the Bearer token and re-checks workspace ownership of the session.**

Event types (v1):

- `session.status` — `{status, reason?}`. Emitted on every transition.
- `message.start | message.delta | message.stop` — assistant text streaming. Mirrors Anthropic streaming shape.
- `tool_use` — `{tool_use_id, tool_name, tool_type, input}`. For `builtin`/`mcp`: informational, a `tool_result` follows. For `custom`: session → `waiting_for_tool`, client must post a `tool_result` via `/messages` to resume.
- `tool_result` — `{tool_use_id, content, is_error}`. Emitted after server-side execution, or echoed after the client posts one.
- `cost.update` — `{provider, session_total_tokens, session_total_usd, last_call: { /* provider-tagged: anthropic = input/output/cache_read/cache_create; openai = input/output/reasoning_tokens/audio_tokens/image_tokens; gemini per its own fields */ }}`.
- `ask_user` — `{prompt_id, question}`. Session → `waiting_for_input`; resume with `{type: "user_message", in_reply_to: prompt_id}`.
- `error` — `{code, message, retryable}`. Codes include `tool_validation`, `session_budget_exceeded`, `rate_limited`, `provider_error`. Terminal unless retryable.
- `ping` — keepalive emitted only after 15s of stream silence (LB idle-timeout defense without active-stream noise).

**Event IDs are ULIDs.** Sequential IDs are not used; reconnect cannot enumerate other sessions' events.

**Split:** SSE is server→client only. All client→server input goes through POST endpoints. Matches what works cleanly behind AWS ALB/NLB.

## Quotas and rate limits

Per-workspace defaults (override via workspace settings):

| Limit | Default |
|---|---|
| Sessions created / minute | 60 |
| Concurrent active sessions | 100 |
| Max message size | 1 MB |
| Concurrent SSE connections per API key | 50 |
| Skill bundle storage per workspace | 1 GB |
| Per-session `max_tokens` ceiling | unset (configurable per session) |
| Per-session `max_usd` ceiling | unset (configurable per session) |

Exceeding a session ceiling emits `error` with `code: "session_budget_exceeded"` and terminates the session. Rate-limit rejections return `429` with `Retry-After`.

## Durable execution (Enterprise) — DBOS

Each session = one DBOS workflow. Loop iterations, LLM calls, and tool invocations are DBOS steps. Steps are checkpointed to Postgres before execution, results persisted after. On pod crash, DBOS replays the workflow, skips completed steps, and resumes at the first unfinished one.

- **LLM call** → DBOS step. Response persisted; replay returns stored response (no re-billing).
- **Built-in tool call** → DBOS step. Idempotency key = `session_id:tool_use_id`. Non-idempotent built-ins (e.g. `exec`) commit intent → execute → commit result.
- **MCP tool call** → DBOS step. Idempotency key forwarded as `Idempotency-Key` header when supported.
- **Custom tool** → DBOS workflow yields via durable `recv()`; resumes on `POST /messages` with `tool_result`. Wait survives pod restarts.
- **Event emission** → written to `SessionStore` inside the same Postgres transaction as the DBOS step commit; SSE replay and workflow state cannot diverge.
- **Ownership/recovery** → DBOS recovery sweeper reclaims orphaned workflows on pod death. No bespoke lease logic.
- **Client-side idempotency** → `Idempotency-Key` on `POST /sessions` and `POST /sessions/{id}/messages` deduplicates retries inside a 24h window via a Postgres-backed key→response cache. A retried `tool_result` POST returns the original session-state delta — duplicate submissions cannot cross the durability boundary.

Lite does not run DBOS. Crash = loss of in-flight step; documented as an intentional tier difference. API contract is identical across tiers; the only observable difference is crash-recovery behavior.

## Auth, multi-tenancy, and Lite hardening

- **Auth**: API keys scoped to a workspace. `Authorization: Bearer <key>`. Lite loopback bind accepts any key (true zero-config dev). **Lite refuses to bind non-loopback addresses unless `OPENHARNESS_DEV_KEY` is set; once set, Bearer auth is enforced and the dev key is the only accepted credential.**
- **Tenancy**: single control-plane Postgres partitioned by `workspace_id` derived from the API key. Per-tenant k8s namespaces are a deployment option, not a v1 requirement.
- **OIDC/SSO** for the admin dashboard is a separate workstream, not part of the SDK spec.

## Retention and right-to-delete

- **Soft-delete**: `DELETE /sessions/{id}` marks the session deleted. PII-scrub job redacts `message.delta` content and tool inputs/outputs after 30 days (workspace-configurable). Cost/metric rows preserved with PII-free fields only.
- **Workspace RTBF**: `DELETE /v1/workspaces/{id}/data` enqueues an async job that hard-deletes all sessions, agents, versions, secrets, and audit events for the workspace. Returns a job handle; `GET /v1/workspaces/{id}/data/jobs/{job_id}` reports progress. SLA: 30 days.
- **DBOS workflow state** for deleted sessions is purged in the same job.

## Model and provider keys

- Canonical model identifier: `provider/model`, e.g. `anthropic/claude-sonnet-4-6`, `openai/gpt-4o`, `google/gemini-2.0-pro`. Bare identifiers are rejected at agent create.
- `GET /v1/models` lists supported models per workspace, including which require BYO keys.
- `model_provider_key` on agent: omitted → `platform` (openharness pays, billed back at platform rate); `${secret:my_anthropic_key}` → BYO via Secrets resource.

## Go SDK surface

Three modules:

- `github.com/swiftdiaries/openharness-sdk-go` — types, errors, transport interface. No framework deps.
- `github.com/swiftdiaries/openharness-sdk-go/httpclient` — HTTP+SSE transport. No framework deps.
- `github.com/swiftdiaries/openharness-sdk-go/inprocess` — in-process transport. Depends on framework.

```go
import (
    "github.com/swiftdiaries/openharness-sdk-go"
    "github.com/swiftdiaries/openharness-sdk-go/httpclient"
    // or "github.com/swiftdiaries/openharness-sdk-go/inprocess"
)

c, _ := openharness.NewClient(httpclient.New("https://...", "sk-..."))
// c, _ := openharness.NewClient(inprocess.New(runner))

c.Agents.Create / Get / List / Update / Archive
c.Agents.Versions.List / Get
c.Agents.Run(ctx, agentID, message) (EventStream, error)  // one-liner: creates session, returns stream
c.Sessions.Create / Get / List / SendMessage / Cancel / Interrupt / Delete
c.Skills.Upload / List / Get / Archive
c.Secrets.Create / List / Delete
c.Webhooks.Create / List / Delete
c.AuditEvents.List
c.Models.List

stream, _ := c.Sessions.Events(ctx, id, EventsParams{LastEventID: "..."})
for stream.Next() { ev := stream.Event() /* typed */ }

c.Sessions.Run(ctx, id, openharness.Handlers{
    OnText:    func(text string) { ... },
    OnAskUser: func(prompt openharness.AskUser) (string, error) { ... },
    OnToolUse: openharness.ToolHandlers{
        "check_inventory": func(ctx, input) (any, error) { ... },
    },
    OnCost:    func(cost openharness.Cost) { ... },
    OnError:   func(err error) { ... },
})
```

**Typed errors**: `errors.Is(err, openharness.ErrRateLimited)` works for `ErrRateLimited`, `ErrUnauthenticated`, `ErrNotFound`, `ErrInvalidRequest`, `ErrToolValidation`, `ErrSessionBudgetExceeded`, `ErrServer`. SSE `error` events are mapped to the corresponding sentinel.

**Transport interface:** small `Transport` abstraction (`Do(req) (resp, stream, err)`) implemented by `httpclient` and `inprocess`. HTTP uses `net/http` + `bufio`-based SSE parser. InProcess adapts `HarnessRunner.Dispatch` + framework event channel to the SDK's typed event stream.

## Non-Go clients — codegen plan (deferred)

- Spec artifacts committed in the openharness repo: `api/openapi.yaml` (control plane) and `api/events.md` + `api/events.schema.json` (SSE event schema). `input_schema` and `output_schema` fields use **JSON Schema 2020-12**.
- Generator of record: openapi-generator for REST; event schema codegen'd via `quicktype` or hand-rolled thin types per language.
- Reserved package names: `openharness` (PyPI), `@openharness/sdk` (npm).
- Trigger to build: (a) paying Enterprise tenant or external vertical asks for it, or (b) an internal non-Go repo needs a client. Until then, the spec artifacts are the only deliverable.

## Deployment (Enterprise)

- **Workloads:** stateless `openharness-api` deployment (HTTP+SSE), `openharness-worker` deployment (DBOS workflow executors). Same binary, different entrypoints.
- **Datastore:** Postgres (DBOS state + SessionStore events + agents/versions/skills/secrets metadata + audit log). Managed Postgres recommended.
- **KMS:** AWS KMS / GCP KMS / HashiCorp Vault for the workspace KEK that wraps Secrets DEKs. Configured per deployment.
- **Egress proxy:** required; all MCP outbound traffic transits it for the controls in the MCP egress section.
- **Object store:** S3-compatible for skill bundles and artifacts.
- **Ingress:** AWS ALB (or any HTTP LB) with SSE idle timeout ≥ 300s; `ping` keepalives only when the stream is silent for 15s.
- **Scaling:** API pods on request rate; worker pods on DBOS queue depth.
- **Helm chart:** `deploy/helm/openharness-enterprise` — separate deliverable, referenced not designed here.

## Milestones

- **v1 (Layer 6 release):** HTTP+SSE API, Go SDK (three modules, both transports), DBOS durable execution in Enterprise, Agents/Versions/Sessions/Skills/Secrets/Webhooks/AuditEvents, tools (builtin/mcp/custom), Sandbox admission rule for `exec`, workspace-scoped Bearer auth, Lite hardening, MCP egress controls, retention/RTBF, per-session budgets, JSON Schema 2020-12, OpenAPI + event-schema artifacts.
- **Depends on:** Layers 1–5 (see spec header for Lite-only-slice carve-out).
- **Post-v1:** TS/Python SDKs via codegen (on demand), Environments, Files/Artifacts, OIDC/SSO, code-executing skill bundles (riding on Sandbox), Helm chart hardening.

## Verification

End-to-end checks once implemented:

1. Create agent → create session → stream events → assistant streams text → session completes. Both transports.
2. Custom tool round-trip: server emits `tool_use` for a `type:"custom"` tool; client posts `tool_result`; session resumes and completes. Server rejects `tool_result` whose content violates the declared `output_schema`, emitting `error` with `code: "tool_validation"`.
3. Durable execution: kill API pod mid-LLM-call; session resumes on another pod without re-billing and without data loss. Kill worker pod mid-tool-call; tool is not re-executed if idempotent result was committed. Retried `POST /sessions/{id}/messages` with the same `Idempotency-Key` returns the original response, never duplicates `tool_result`.
4. SSE resume: drop the client connection mid-stream; reconnect with `Last-Event-ID`; missed events replayed. Reconnect with a Bearer token that no longer owns the session is rejected with `401`.
5. Lite parity: same Go SDK code path works against `inprocess.New(runner)` and against a local `openharness serve` — identical observable behavior except crash recovery. `openharness serve --bind 0.0.0.0` without `OPENHARNESS_DEV_KEY` set refuses to start.
6. Auth: requests without Bearer token rejected on Enterprise; cross-workspace data access impossible via API key scoping.
7. MCP egress: agent declaring an MCP URL pointing at `169.254.169.254` or `10.0.0.1` is blocked at call time; response over 10MB is truncated and emitted as `error`; per-call timeout fires at `timeout_seconds` (or 120s default, 600s ceiling).
8. Sandbox admission: agent with `exec` in tools and `trust_mode: "restricted"` is rejected at create with `400 invalid_tool_config`. Same agent with `trust_mode: "safe"` and `sandbox.type: "runloop"` is accepted; with `sandbox.type: "lambda"` and not on the deployment allowlist, rejected.
9. Secrets: MCP header value `${secret:my_token}` resolves at call time; plaintext header values trigger an audit event; non-loopback Lite refuses plaintext.
10. Per-session ceiling: session created with `max_usd: 0.50`; agent loops until cost exceeds; session emits `error` with `code: "session_budget_exceeded"` and terminates.
11. Audit log: creating an agent and a secret each produces a corresponding `AuditEvents` row; `GET /v1/audit_events` paginates correctly.
12. Webhook: subscribed to `session.completed`; running session to completion delivers a signed POST; `X-Openharness-Signature` verifies against the per-webhook secret.
13. Retention: `DELETE /sessions/{id}` then advance through scrub interval (compressed in test); message content redacted, metric rows present.

## Adversarial review (2026-04-18)

Original critique with resolution status. Items marked **[resolved]** are addressed in the spec above.

### Security
- **S1** [resolved]: Lite refuses non-loopback bind without `OPENHARNESS_DEV_KEY`; loopback bind accepts any key.
- **S2** [resolved]: Egress proxy + RFC1918/IMDS deny + TLS verify required + 10MB cap + 120s default / 600s max timeout.
- **S3** [resolved]: Minimal `Secrets` resource added (KEK-wrapped per-workspace DEK; AWS KMS / GCP KMS / Vault).
- **S4** [resolved]: Server validates `custom` tool inputs and outputs against JSON Schema 2020-12 schemas.
- **S5** [resolved]: `exec` gated on (sandbox, trust_mode) admission rule; default off; agent-create rejects misconfig.
- **S6** [resolved]: v1 skill bundles are declarative-only; code-executing bundles deferred and inherit Sandbox rule.
- **S7** [resolved]: Per-workspace defaults specified in Quotas section.
- **S8** [resolved]: ULID event IDs; SSE reconnect re-validates Bearer + session ownership.
- **S9** [resolved]: `Idempotency-Key` required on `POST /sessions` and `POST /sessions/{id}/messages`; 24h dedup.
- **S10** [resolved]: `AuditEvents` resource added.

### Ease of use
- **U1** [resolved]: `c.Agents.Run(ctx, id, message)` one-liner.
- **U2** [resolved]: Three Go modules; HTTP-only consumers don't pull framework deps.
- **U3** [resolved]: Unified `Handlers{OnText, OnAskUser, OnToolUse, OnCost, OnError}`.
- **U4** [resolved]: Sessions snapshot `latest_version` at create; `auto_advance` is opt-in.
- **U5** [resolved]: JSON Schema 2020-12 specified.
- **U6** [resolved]: cancel = terminal; interrupt = stop step, session resumable.
- **U7** [resolved]: `cost.update` is provider-tagged.
- **U8** [resolved]: `POST /sessions?stream=true` returns SSE in same response.
- **U9** [resolved]: Typed Go errors enumerated.
- **U10** [resolved]: `provider/model` canonical IDs; `GET /v1/models`.
- **U11** [resolved]: `ping` only after 15s of stream silence.

### Missing surfaces
- **Retention/RTBF** [resolved]: `DELETE /sessions/{id}` + workspace RTBF job; PII-scrub.
- **Webhooks** [resolved]: `Webhooks` resource + HMAC-SHA256 delivery.
- **Per-session ceiling** [resolved]: `max_tokens`, `max_usd` on Session.create.
- **Model-provider-key scoping** [resolved]: `model_provider_key` on agent (`platform` default vs `${secret:...}` BYO).
