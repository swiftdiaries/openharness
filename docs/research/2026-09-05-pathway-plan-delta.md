# Existing plans versus the proposed agent pathway

Date: 2026-09-05. Status: scope assessment, not an approved plan amendment. Compared with the [pathway including evaluation](2026-09-05-declarative-agents-and-sandboxes.md) at OpenHarness `6fc0756`. There were no Git-staged files when this assessment began. “Staged plans” here means the recorded OpenHarness plans and Kiteframe's staged roadmap, including explicitly identified drafts.

## What changes overall

The pathway **retains the Go primitives and declarative composition plan, changes the first delivery milestone, and adds a cross-project execution and evaluation path**. It is not a replacement specification for all six OpenHarness layers.

The user clarified the direction during this assessment: **remove required pieces such as the DBOS session runtime from the delivery scope, preserving an interface and explicit stub for a future implementation.** OpenHarness can supply a bounded harness to Kiteframe without first implementing a second durable session engine. This records the requested planning direction; the existing SDK specification still needs a focused amendment before implementation.

“Not implemented” is not the same as “new scope.” OTel storage, sandbox admission, named drivers, profiles and YAML already appear in existing designs. The new work is their specific integration, earlier delivery and additional acceptance criteria.

## First-pass scope after user clarification

The user requested a minimal design with deep modules and explicit stubs, followed by later implementations. See the [first-pass module design](2026-09-05-declarative-agents-and-sandboxes.md#minimal-first-pass-deep-modules-with-explicit-stubs).

- **Build first:** one bounded OpenHarness loop/tool/result path, real local trace capture/export and one deterministic result check, through small existing interfaces wherever possible.
- **Reserve with explicit stubs:** durable session recovery, sandbox execution and unavailable evaluator adapters. Stubs reject unsupported operations before effects; they do not simulate production guarantees.
- **Fill next:** the approved Kiteframe worker/sandbox adapter, optional OpenHarness ACP integration, browser review and broader eval comparison. DBOS is optional future implementation work.
- **Keep internal:** loop helpers, provider-specific mechanics and storage wiring that callers do not need to configure independently. “Extensible” does not require a public interface for each helper.

The packages below describe the full pathway delta, not a requirement to build them all in pass one. The first local proof is explicitly non-isolated and non-durable; a sandboxed/recoverable product claim requires the later real adapters and their acceptance tests.

## Baseline and its limits

| Source | What is actually recorded |
|---|---|
| [Layer 2 execution order](../superpowers/plans/layer-2/2026-04-13-execution-order.md) | Plans 1–3 landed; remaining waves describe MCP, loop, app registration, Ghostfin migration and scaffolder. Detailed Plan 4–8 files are referenced as future documents and are not present in this checkout. |
| [Platform composition proposal](../../openspec/changes/platform-builder-runtime/proposal.md), [design](../../openspec/changes/platform-builder-runtime/design.md), [tasks](../../openspec/changes/platform-builder-runtime/tasks.md) | Detailed active change for eight bindings, Lite/Enterprise profiles, driver registries, YAML/JSON and adoption. Implementation tasks remain unchecked. Its completed research tasks do not signify implementation. |
| [Extraction design](../superpowers/specs/2026-04-10-openharness-extraction-design.md) | Layer 1.5 event/input enrichments and broad Layers 3–5 architecture; these are not all detailed execution plans. |
| [Layer 6 SDK design](../superpowers/specs/2026-04-17-layer-6-sdk-design.md) | Designed HTTP/SSE/in-process client surfaces, sandbox admission, DBOS-backed Enterprise sessions, authentication and recovery acceptance. No implementation plan is present. |
| Kiteframe local main and cumulative worktree | Local main contains Slices 1–4. The cumulative design stages later work behind production-spec gates; Slice 5's local plan remains decision-gated. This does not authorize a sandbox provider or OpenHarness integration. |

The OpenHarness extraction and Layer 2 specs are marked Draft even where portions have shipped. Treat individual committed implementations and acceptance records as delivered evidence; do not infer approval of every future section from the existence of a spec.

## OpenHarness scope delta

| Area | Existing plan | Pathway difference | Classification |
|---|---|---|---|
| Provider clients, tools and agent loop | Layer 2 Plans 1–5 | Reuse them for the first bounded vertical | **Keep.** A new task example is small additional fixture work, not a new agent engine. |
| Event input and interactive control | Layer 1.5 proposes `EventStream.Send`, input-enabled LoopFactory and permission/user-response events | ACP needs bidirectional input, cancellation and exact permission correlation | **Reuse and reconcile.** Do not build a second Go input contract just for ACP. Verify which existing enrichments it needs. |
| App registration | Plan 6 introduces App accessors/registration | Still precedes full composition | **Keep.** The first direct harness demo can be smaller than the complete app scaffold. |
| Profiles, drivers and YAML | `platform-builder-runtime` PRs 1–4 | Same assembly mechanism | **Keep.** Sandbox/eval configuration must not be silently inserted into the existing eight-binding schema. |
| Minimal example and scaffolder | Plan 8 plus composition PR-5 | Pathway wants a useful bounded example earlier | **Resequence.** An early internal fixture does not replace the published-module/scaffolder acceptance. |
| Ghostfin adoption | Plan 7 and existing release gates | Not required to prove the Kiteframe integration | **Separate milestone.** Existing Ghostfin compatibility/release promises remain unless explicitly amended. |
| OTel trace storage | Layer 3 already describes metadata store, span sink and local/OTLP destinations | Capture useful traces alongside the first loop; add context propagation, linked attempts and completeness | **Bring forward and extend.** Not a new observability architecture from zero. |
| TraceRecorder and composition bindings | Recorder interface exists; task 9.2 defers concrete telemetry bindings | Start/end contexts for nested spans, payload capture policy and explicit disabled/lost capture | **Amend contract in a separate change.** Do not expand the composition change's explicit non-goals implicitly. |
| OpenHarness ACP server | Existing external consumer surface is Layer 6 HTTP+SSE | Add a Go ACP server that Kiteframe can invoke | **New transport and conformance work.** This is not already provided by HarnessRunner. |
| Sandbox selection and policy | SDK has Sandbox type and `exec` admission rules | Concrete provider experiment, deployment-owned execution profile and worker-controlled lifetime | **Partial overlap, substantial new integration.** Reconcile authority and schema before coding. |
| Enterprise execution | Layer 5 K8s/Lambda runners; SDK DBOS session workflows | Use a bounded OpenHarness invocation under Kiteframe; evaluate OSS compute | **Reduce required scope.** Remove the DBOS implementation requirement from the proposed delivery path; retain a small runtime seam and unsupported stub. Amend SDK durability promises separately. |
| Desktop UI and SDK | Layers 4/6 and Lite-only SDK carve-out | Not prerequisites for the patch/test milestone | **Defer from this milestone only.** These were already separate layers. |
| Review interface | No corresponding planned application | Whole-case review, Pass/Fail/Defer, notes, persistence and artifact display | **New subsystem.** Separate from operational trace browsing and the app-composition GUI non-goal. |
| Eval cases and comparison | Existing unit/integration/recovery tests; trace design mentions eval use | Versioned cases, rubric/labels, evaluators, baseline/candidate comparison and later calibrated judges | **New workflow built on existing tests.** Existing deterministic gates remain. |

Sources for the specific contracts: [Layer 1.5](../superpowers/specs/2026-04-10-openharness-extraction-design.md#layer-15-adoption-enrichments-v020), [telemetry storage](../superpowers/specs/2026-04-10-openharness-extraction-design.md#trace-instrumentation--two-layer-architecture), [SDK sandbox](../superpowers/specs/2026-04-17-layer-6-sdk-design.md#sandboxes), [SDK durability](../superpowers/specs/2026-04-17-layer-6-sdk-design.md#durable-execution-enterprise--dbos).

## Kiteframe scope delta

Fresh local inventory: main `06c3abb5aa9385e9b76c5d47e754afa7df6e49de` contains Slices 1–4; the unmerged cumulative worktree is at `f7db393b8b285f184accfc6121145ca5b78949e4` with additional uncommitted release fixtures. Its board has tickets 1–3 resolved, 4 claimed and 5–81 open. These local revisions are not claims of upstream delivery.

| Area | Existing staged scope | Pathway delta |
|---|---|---|
| Work authority and recovery | Cumulative design already requires authorization, worker leases/fences, mutation barriers, uncertainty, reconciliation and kernel-owned completion | **Preserve.** These are not new sandbox features invented by the pathway. Production Slices 5–8 still require their own approved specifications. |
| Worker execution | Slice 5 draft proposes provider-neutral worker delivery around existing harness envelopes | **Extend.** Worker becomes the sole sandbox broker: provision, launch, reconcile, clean up and enforce provider capabilities. Current host-process ACP launch is not sandbox isolation. |
| Harness profiles | Current/deterministic ACP profiles and in-process built-in Responses harness | **Add optional integration.** OpenHarness ACP is a new profile/transport, not a V1 dependency. Sandboxing the built-in in-process harness would be separate work. |
| Execution policy | Slice 5 draft D13 allows minimum worker/workspace/capability/executor-profile identity; richer governance belongs to Slice 6 | **Reconcile scope boundary.** Digest-pinned images, egress, credentials and resources expand that minimum. Settle which enforcement is required in Slice 5 and which policy authoring remains Slice 6. |
| Release receipts | Cumulative durable receipt ledger proves release gates | **Add observational tooling.** OTel, review bundles, labels and eval manifests are new; none grants work authority or completes goals. |
| End-to-end product delivery | Production adapters and complete babysit-pr flow in Slices 7–8 | **Keep staged.** A patch/test sandbox experiment cannot replace cumulative release acceptance. |

Sources: the [approved cumulative design](https://github.com/swiftdiaries/kiteframe/blob/6b0ca0e/docs/superpowers/specs/2026-08-30-babysit-pr-cumulative-golden-thread-design.md), especially “Production Seam,” “Two Required Gates,” and Slice 6; and the [local Slice 5 draft](/Users/adhita/projects/python/src/github.com/swiftdiaries/kiteframe/docs/superpowers/plans/2026-08-30-kiteframe-declarative-agents-slice-5.md), especially D1/D12/D13, “Authority and ownership,” and its crash/recovery matrix. The latter is untracked and decision-gated, not an approved specification.

## Decisions that must be reconciled

### 1. Replace the required DBOS implementation with an extension seam

**User-directed scope reduction:** the SDK design currently requires each Enterprise session to be a DBOS workflow, with checkpointed model/tool steps and orphan recovery. Remove that required implementation from the proposed pathway. Keep a small session-runtime interface and an explicit stub so a durable backend can be added later. DBOS becomes a possible implementation, not an architectural dependency.

The seam should describe only operations required by the first actual consumer. Reuse the existing HarnessRunner boundary where sufficient; introduce a separate session-runtime contract only for demonstrated session orchestration needs. Do not encode DBOS workflow IDs, database tables or checkpoint mechanics in the public contract. The exact signatures remain a specification task.

The stub must report unsupported durable execution/resume/recovery clearly at configuration or invocation time. It must not acknowledge durable acceptance, silently fall back to an in-memory runner, or imply that persisted transcripts provide crash recovery. Existing Lite execution can remain explicitly non-durable. Any future backend must declare supported capabilities and satisfy the same lifecycle/recovery contract tests before being selected.

In the Kiteframe integration, Kiteframe retains work authorization and recovery ownership; OpenHarness performs one bounded harness invocation. A future durable session plugin must not independently reassign or retry the same external effect without an agreed ownership contract.

**Concrete plan amendments needed:** remove DBOS as a mandatory SDK/Enterprise dependency; replace workflow/sweeper delivery tasks with interface/stub and capability-rejection acceptance; mark crash-resume, checkpoint replay and durable stream recovery promises as deferred until a real backend passes tests. Keep HTTP/SSE, session identity and transcript storage only to the extent they work without those guarantees. This is a scope reduction, not a request to implement Kiteframe as a replacement session engine.

### 2. Agent configuration versus deployment authority

The SDK design allows agent-level overrides of workspace `sandbox` and `trust_mode` defaults and admits `exec` using a fixed sandbox-type list. The pathway introduces deployment-owned, digest-pinned execution profiles and requires verified enforcement. OpenSandbox/Substrate do not fit that fixed list directly; some existing entries name isolation mechanisms while others name execution services.

Decide how an agent requests a permitted profile while deployment policy limits image, resources, credentials and egress. Define the effective-policy calculation and unsupported-capability rejection. Merely adding a new provider name to the SDK enum does not implement isolation or settle override authority. Keep the standalone SDK admission API and the Kiteframe-hosted policy consistent without making their identity models identical.

### 3. Delivery order versus existing release promises

The current wave order puts MCP before the loop, app registration next, then the published-module example/scaffolder and Ghostfin adoption. The pathway's first proof can use a direct loop with existing tools and an internal fixture. That is a new intermediate milestone; it is not completion of Plan 8 or `v0.2.0`.

Similarly, the pathway brings minimal real trace capture forward from Layer 3. Preserve the full local metadata/span storage design as later work unless there is an explicit decision to change it. Do not require the full trace analytics backend to open the first review bundle.

### 4. “Every layer is extensible” remains a separate gap

The current composition design explicitly excludes dynamic plugin loading, RPC plugins, changes to the eight harness interfaces and an LLM-provider profile axis. The pathway adds selected execution, telemetry and eval boundaries; it does **not** supply a complete extension contract for every layer.

If every meaningful layer must be replaceable, the next design needs a component-by-component matrix: contract, registration/configuration, lifecycle, compatibility and permissions. The agent loop, scheduler/app lifecycle and some infrastructure remain outside the eight-binding model. This does not automatically require installable plugins or hot reload. An OpenHarness Go ACP server is a new consumer transport; it does not mean importing a non-Go runner into OpenHarness or reversing the Go-native authoring choice.

## Recommended work packages and dependencies

These are proposed scope boundaries, not a new task board. Approved work should be recorded in the owning repository's existing process.

| Package | Owner and prerequisites | Bounded result |
|---|---|---|
| Existing runnable foundation | OpenHarness; remaining Layer 2 and relevant Layer 1.5 contracts | One real loop/tool/artifact/cancellation example. Keep composition on its Plan 6 dependency. |
| Trace capture and review export | OpenHarness recorder plus Kiteframe adapter; useful execution boundary | OTel spans, explicit correlation/capture status and an immutable JSON/CSV review bundle. Full analytics storage is not required for the first export. |
| Human review and eval cases | OpenHarness tooling; stable review bundle | Local review UI/SQLite annotations, deterministic case checks and baseline/candidate comparison. No live-goal mutations or automatic judge gate initially. |
| Guarded sandbox attempt | Kiteframe's approved Slice 5 design; effective execution-policy decision | One worker owns provisioning, ACP stream transport, result reconciliation and cleanup on one provider. Integrate with planned Slice 6 policy rather than duplicating it. |
| OpenHarness ACP integration | Runnable Go harness plus stable worker transport/profile contract | Portable ACP server/profile; compare with one existing out-of-process ACP profile. The built-in in-process harness stays on its existing path. |
| Optional scale/backend work | Evidence from the first provider | Evaluate Substrate/AX or another backend only against a demonstrated need; no requirement to adopt all candidates. |

The earliest independent work is the runnable OpenHarness foundation and trace-contract design while Kiteframe completes its predecessor checks. Review tooling can start from validated example bundles, but its acceptance requires real captured runs. The guarded sandbox path depends on a Slice 5 decision; the optional OpenHarness integration must not block the existing Kiteframe V1 release.

## What this assessment does not change

No production spec, implementation checkbox or release gate is approved or rewritten here. The user-directed DBOS reduction is recorded here for a subsequent specification amendment; no runtime code was changed. Migrating Ghostfin, installing sandbox infrastructure, publishing plugins and changing Kiteframe's accepted scope are not implied. The concrete next planning action is to specify the minimal first pass, amend the SDK runtime requirement, settle policy ownership for the later guarded path, and create focused changes for new work rather than adding everything to `platform-builder-runtime`.
