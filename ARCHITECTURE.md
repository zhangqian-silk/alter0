# Alter0 Architecture (Target Design)

## 1. Design Goals

Alter0 is a self-hosted, single-instance orchestration kernel for developers.  
The target architecture focuses on:

1. Task-first orchestration instead of chat-thread-first orchestration.
2. Stable 24x7 runtime on a single node with clear recovery behavior and repeatable deployment assets (systemd/Docker).
3. Capability isolation: reusable runtime capabilities are hosted in `app/pkg`, while business orchestration stays in `app/core`.
4. Strong abstraction boundaries so modules evolve independently.
5. Observable and maintainable operations without introducing unnecessary platform complexity.

## 2. Boundary and Non-goals

- Deployment model: developer self-hosted, trusted local/LAN environment.
- No account system or full multi-tenant control plane in core architecture.
- No mandatory centralized auth stack or internet-facing anti-abuse gateway.
- Default topology remains single-instance; distributed mode is optional future extension.

## 3. Layered Architecture

```text
+---------------------------------------------------------------+
| Interaction Layer                                             |
|  CLI / HTTP / Web / Future: IM, IDE, Webhook                 |
+------------------------------+--------------------------------+
                               |
                               v
+---------------------------------------------------------------+
| Gateway Layer                                                 |
|  Session Context, Message Envelope, Routing to Orchestrator   |
+------------------------------+--------------------------------+
                               |
                               v
+---------------------------------------------------------------+
| Orchestration Layer                                           |
|  Minimal Workflow Core: resolve -> execute -> close           |
+------------------------------+--------------------------------+
                               |
                               v
+---------------------------------------------------------------+
| Capability Modules (app/pkg + app/core)                      |
|  queue/scheduler/logger + runtime/tools/schedule/task         |
+------------------------------+--------------------------------+
                               |
                               v
+---------------------------------------------------------------+
| Infrastructure Layer                                          |
|  SQLite, local FS logs/audit, external channels, executors    |
+---------------------------------------------------------------+
```

## 4. Core Domain Model

### 4.1 Primary Entities

- `Task`: lifecycle container of work.
- `Message`: task-scoped interaction record.
- `Execution`: one model/tool execution attempt.
- `Artifact`: generated external outputs (optional extension).
- `TaskMemory`: compact context snapshot.
- `Plugin`: runtime-loaded capability unit (`skill` or `mcp` connector).

### 4.2 Task State Machine

`open -> running -> waiting_input -> running -> closed`  
Optional transient states: `failed`, `cancelled`, `retrying`.

### 4.3 Decision Chain

Task selection priority:

1. forced task (`/task use`)
2. explicit `task_id`
3. model router decision (`existing`/`new`)
4. fallback new task

Task close decision:

- model close decision + confidence threshold + policy guard

## 5. Module Architecture

### 5.1 Interaction Layer

Responsibilities:

- normalize inbound requests into `MessageEnvelope`
- expose consistent sync/async interfaces
- maintain channel-local UX and transport concerns only

Current baseline (implemented):

- `types.Message` now carries transport-neutral `Envelope`
- `Envelope.Parts` uses typed units (`text/image/audio/file/reference`) for cross-channel normalization
- CLI/HTTP adapters populate inbound envelope fields, and gateway normalizes outbound envelope direction
- Telegram (long polling) + Slack (Events API) adapters provide external channel baseline with media-capable outbound envelope mapping

Required modules:

- `interaction/cli`
- `interaction/http`
- `interaction/telegram`
- `interaction/slack`
- `interaction/web`
- `interaction/<future-channel>`

Abstraction contract:

```go
type InteractionChannel interface {
    ID() string
    Start(ctx context.Context, h InboundHandler) error
    Send(ctx context.Context, out OutboundMessage) error
    Health() ChannelHealth
}
```

### 5.2 Gateway Layer

Responsibilities:

- channel registration and lifecycle
- message dispatch and reply correlation
- multi-agent route resolution (`agent_id` -> registry, default fallback)
- lightweight health aggregation and crash containment

Gateway does not own task policy or execution policy.

Current baseline: registry-backed agents are declared by `agentId/workspace/agentDir`; each agent keeps an isolated task store while sharing the same channel gateway process.

### 5.3 Orchestration Layer

Responsibilities:

- preserve only minimal orchestration logic (`route -> execute -> close`)
- manage orchestration state transitions and decision chain only
- invoke capability interfaces (`app/pkg` + `app/core/runtime`), without owning persistence/runtime infrastructure

Required submodules:

- `orchestrator/agent` (main flow coordinator)
- `orchestrator/task` (router + closer + core task model)
- `orchestrator/policy` (target-state independent policy pack)
- `orchestrator/context` (prompt/memory builder)

### 5.4 Capability Modules (`app/pkg` + `app/core`)

Responsibilities:

- host independent reusable capabilities outside orchestration core
- provide stable service contracts to gateway/orchestrator
- centralize runtime ownership of store/queue/scheduler/log/audit/status

Required modules:

- `pkg/queue` (job queue runtime and retry controls)
- `pkg/scheduler` (job scheduling runtime)
- `pkg/logger` (shared logger)
- `core/orchestrator/schedule` (at/cron center + delivery)
- `core/orchestrator/task` (task persistence + memory + task stats)
- `core/orchestrator/command` (slash/admin command execution and permission checks)
- `core/runtime` (preflight, maintenance, lifecycle integration)
- `core/runtime/tools` (tool policy/runtime/audit bridge)

Layer boundary rule:

- `core/orchestrator` may depend on `pkg/*` reusable capabilities.
- `pkg/*` must not depend on `core/orchestrator`.

### 5.5 Execution Layer (Agent Runtime)

Responsibilities:

- abstract over different agent executors (`codex`, `claude_code`, future providers)
- uniform execution request/response model
- timeout, cancellation, retry, structured error mapping

Abstraction contract:

```go
type AgentExecutor interface {
    Name() string
    Check(ctx context.Context) error
    Execute(ctx context.Context, req ExecRequest) (ExecResult, error)
}
```

### 5.6 Extension Layer (Skill + MCP)

This layer is first-class in the target architecture.

#### Skill Host

- loads local or packaged skill plugins
- validates manifest/version compatibility
- exposes typed invocation API to orchestrator/executor

```go
type Skill interface {
    Manifest() SkillManifest
    Invoke(ctx context.Context, call SkillCall) (SkillResult, error)
}
```

#### MCP Bridge

- manages MCP server connections (local process or remote endpoint)
- maps MCP tools/resources/prompts into unified capability registry
- handles capability discovery, timeout, retry, and connection health

```go
type MCPConnector interface {
    ID() string
    Connect(ctx context.Context) error
    Discover(ctx context.Context) ([]Capability, error)
    Invoke(ctx context.Context, call CapabilityCall) (CapabilityResult, error)
    Close(ctx context.Context) error
}
```

#### Unified Capability Registry

All callable capabilities are registered under one runtime index:

- built-in skill
- external skill plugin
- MCP tool/resource adapter

This enables policy-driven tool selection without coupling to source type.

#### Tool Protocol Baseline (Implemented)

Gateway runtime now ships a normalized tool protocol for:

- `web_search`
- `web_fetch`
- `browser`
- `canvas`
- `nodes`
- `message`
- `tts`
- `memory_search`
- `memory_get`

Each tool invocation passes through a shared policy gate (`global_allow/global_deny/agent allow-deny/require_confirm`) and produces a unified result contract (`success/failed/retryable/blocked` + structured error code). High-risk tools (`browser/canvas/nodes/message`) always require explicit confirmation as baseline guardrails.

For deep toolchain alignment, runtime status also publishes `tools.protocol.toolchain` action schemas for `browser/canvas/nodes`, plus a `live_wiring` map indicating whether each high-risk tool has been wired to a runtime adapter. Runtime rejects unsupported actions / missing required nested fields (for example `browser act` requires `request.kind`, `nodes run` requires `command`, `nodes invoke` requires `invokeCommand`), and can dispatch to registered adapters when invocation callback is omitted.

Long-term memory tools apply an additional context-safety rule:

- trusted channels (`security.memory.trusted_channels`) may access full long-term memory
- shared surfaces are blocked from restricted paths (`security.memory.restricted_paths`, default `MEMORY.md`)

Audit records are appended to `output/audit/<date>/tool_execution.jsonl` with parameter/result summaries and source session metadata. Runtime status (`/status`, `/api/status`) exposes `tools.security_posture` with conflict/widened-scope checks for security review.

### 5.7 Infrastructure Layer

Responsibilities:

- provide concrete infrastructure backends used by `app/pkg` and `app/core` capability modules
- keep infra details isolated from orchestration core

Required modules:

- `infra/store/sqlite` (default)
- `infra/fs` (logs/audit/artifacts)
- `infra/runtime` (maintenance/backup/migration workers)
- `infra/adapters` (external channel/executor/protocol adapters)

## 6. Data Architecture

### 6.1 Core Tables

- `tasks`
- `messages`
- `task_memory`
- `executions`
- `artifacts` (optional but reserved)
- `plugin_registry`
- `schema_meta`

### 6.2 Storage Principles

- append-friendly message/execution logs
- explicit schema versioning with non-destructive migrations
- periodic compacting/retention jobs (closed task retention + optional open task stale-memory cleanup)
- backup-before-migration

## 7. API and Interaction Contracts

### 7.1 Sync API

- `POST /api/message` for short-running interactions
- optional `stream=1` query mode for NDJSON chunked output on long responses

### 7.2 Async API

- `POST /api/tasks` accepts async task requests and returns request id
- `GET /api/tasks` returns recent async requests with status filtering (`status`) and bounded pagination (`limit`)
- `GET /api/tasks/{id}` returns task status (`pending|running|completed|canceled|timeout`), completion payload, and canceled audit fields (`canceled_at/cancel_reason`)
- `POST /api/tasks/{id}/cancel` cancels pending/running async tasks with execution-context interruption semantics
- `POST /api/subagents` creates a sub-agent in `run|session` mode (`run` is async, `session` is long-lived)
- `GET /api/subagents` lists sub-agents with `status/mode/limit` filtering
- `GET /api/subagents/{id}` returns sub-agent state, latest result, and turn history
- `POST /api/subagents/{id}/messages` submits a turn to a `session` sub-agent
- `POST /api/subagents/{id}/close` closes a `session` sub-agent
- sub-agent requests may include `announce.channel_id + announce.to`, which triggers gateway direct delivery after each completed turn
- `GET /api/status` aggregated runtime status (includes schedules/session/subagent/cost metrics, gateway trace summary, risk watchlist snapshot, runtime alerts, recent command audit tail, and git upstream divergence)

### 7.3 Command Surface

- task lifecycle commands
- runtime configuration commands
- plugin/connector management commands

## 8. Runtime Lifecycle

Startup sequence:

1. initialize logger
2. load config
3. run startup preflight (config validity, SQLite writable path, executor binary availability)
4. initialize task/schedule stores and validate schema
5. run migration/backup checks through `core/runtime`
6. initialize executor/tool runtime modules
7. initialize `pkg/queue`, `pkg/scheduler`, and schedule runtime
8. initialize plugin host + MCP bridge
9. initialize orchestration core with capability dependencies
10. start gateway/channels

Shutdown sequence:

1. stop new intake
2. drain queue/scheduler/http with `runtime.shutdown.drain_timeout_sec`
3. close plugins/connectors
4. flush logs/audit services
5. close store/runtime handles

## 9. Observability and Operability

### 9.1 Logs

- structured logs with `session_id`, `task_id`, `stage`, `executor`, `plugin_id`
- dedicated audit streams for command and capability invocations, and status snapshot tail aggregation for quick diagnostics

### 9.2 Metrics

- queue depth, configured workers, in-flight workers, and last shutdown drain report
- schedule status/kind/delivery-mode counters, active overdue windows, and next active run marker
- session and sub-agent activity windows (`total/active/by_channel`) from orchestrator execution logs
- execution latency/error rate plus model usage estimates (`chars/tokens/cost`) with optional pricing config, and `cost.threshold_guidance` global + workload-tier recommendations
- gateway trace window aggregates (`total/error/by_event/by_channel/latest_at`) and derived runtime alerts (`queue_backlog/retry_storm/channel_disconnected/executor_unavailable/session_cost_hotspot/session_compaction_pressure/risk_watchlist_{missing,invalid,stale,item_overdue}`)
- routing/closing decision distribution
- plugin and MCP connector health

Deployment reference assets:

- `deploy/systemd/alter0.service` for long-running host deployment
- `Dockerfile` + `.dockerignore` for containerized runtime packaging

### 9.3 Health Model

- liveness: process alive
- readiness: store + executor registry + plugin host + queue all ready
- degraded: partial dependency failure with fallback path

### 9.4 Console Operations

- Web Console exposes recent async task queue cards backed by `GET /api/tasks`
- operators can refresh statuses, cancel `pending/running` requests, and retry `timeout/canceled` requests without leaving the browser

## 10. Extensibility Rules

### 10.1 Agent Extension

- new agent implementation only needs `AgentExecutor` contract
- no changes required in orchestration core

### 10.2 Interaction Extension

- new channel only needs `InteractionChannel` contract
- gateway auto-registers and uses common envelope

### 10.3 Skill Extension

- plugin package with manifest + version
- registered into capability registry at startup or runtime reload

### 10.4 MCP Extension

- each connector encapsulates transport and protocol details
- orchestrator sees only unified capabilities

### 10.5 Compatibility Policy

- semantic versioning for plugin contracts
- explicit `min_core_version` and `api_version` checks at load time

## 11. Recommended Project Structure (Target)

```text
app/
  configs/
  core/
    executor/
    interaction/
      cli/
      gateway/
      http/
      telegram/
      slack/
    orchestrator/
      agent/
      command/
      execlog/
      schedule/
      skills/
      task/
    runtime/
      tools/
  pkg/
    cmdutil/
    logger/
    queue/
    scheduler/
    types/
```

## 12. Implementation Priorities

P0 (Gateway Foundation):

- N1 multi-channel gateway baseline with deterministic route binding
- N2 multi-agent registry and session isolation
- N3 at/cron scheduling with direct channel delivery
- N6 runtime status expansion (sessions/subagents/schedules/cost)

P1 (Orchestration Upgrade):

- N2 sub-agent run/session lifecycle with announce chain
- N4 normalized tool protocol and policy gates
- N5 long-term memory retrieval and context-safety isolation
- N7 security posture checks and high-risk guard rails

P2 (Advanced Operations):

- N4 deep node/browser/canvas integration
- N6 end-to-end tracing + alerting hardening
- N8 integration matrix and release gate automation

Current status: P0/P1 queues are complete; P2 risk-hardening and P3 parameter governance baseline are merged (N16-N19), and the next delivery candidate is N20 monthly governance cadence automation from a release-gated baseline (`make release-gate`).

## Feature Roadmap

See `docs/features.md` for implemented capability matrix and active queue.
See `docs/openclaw-alignment.md` for versioned OpenClaw parity checkpoints.
See `features.md` for the OpenClaw-aligned backlog of unshipped requirements.

Documentation consistency rule:

- Any PR that updates `docs/features.md` must also update `README.md` and `ARCHITECTURE.md` in the same change set.
- Use `make docs-sync-check` before opening a PR. The check compares `origin/master...HEAD` and fails when only `docs/features.md` changed.
- Use `make release-gate` before merging release-scoped changes; it enforces config-boundary checks (`make check-config-boundary`), service-boundary checks (`make check-service-boundary`), parameter-governance checks (`make config-governance`), test-stability checks (`make test-stability` with hotspot stress + Windows compile regression), integration matrix, risk benchmark gate (`make risk-benchmark`), rollback drill, deployment checks, and doc sync gate in one pass.
