# Alter0 Architecture (Target Design)

## 1. Design Goals

Alter0 is a self-hosted, single-instance orchestration kernel for developers.  
The target architecture focuses on:

1. Task-first orchestration instead of chat-thread-first orchestration.
2. Stable 24x7 runtime on a single node with clear recovery behavior and repeatable deployment assets (systemd/Docker).
3. Pluggable execution/runtime capabilities for `agent`, `interaction`, `skill`, and `mcp`.
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
|  Task Resolver + Workflow Engine + Policy Engine + Commands   |
+------------------------------+--------------------------------+
                               |
             +-----------------+-----------------+
             v                                   v
+------------------------------+   +-----------------------------+
| Execution Layer              |   | Extension Layer             |
|  Agent Adapter Runtime       |   | Skill Host + MCP Bridge     |
+------------------------------+   +-----------------------------+
             |                                   |
             +-----------------+-----------------+
                               v
+---------------------------------------------------------------+
| Data & Runtime Layer                                          |
|  Store, Queue, Scheduler, Observability, Maintenance          |
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

- command handling (`/task`, `/config`, `/executor`, future admin commands)
- task resolve, context assembly, workflow transitions
- policy evaluation (routing, closing, timeout, retry strategy)
- orchestration-level audit logging

Required submodules:

- `orchestrator/agent` (main flow coordinator)
- `orchestrator/command`
- `orchestrator/task` (store + router + closer)
- `orchestrator/policy` (target-state independent policy pack)
- `orchestrator/context` (prompt/memory builder)

### 5.4 Execution Layer (Agent Runtime)

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

### 5.5 Extension Layer (Skill + MCP)

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

### 5.6 Data & Runtime Layer

Responsibilities:

- durable state store
- async execution queue
- scheduler and maintenance jobs
- observability/audit persistence
- backup/migration lifecycle

Required modules:

- `store/sqlite` (default)
- `queue`
- `scheduler`
- `runtime/maintenance`
- `runtime/backup`
- `runtime/migration`

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
- `GET /api/tasks/{id}` returns task status (`pending|completed|canceled|timeout`) and completion payload
- `POST /api/tasks/{id}/cancel` marks a pending async task as canceled
- `GET /api/status` aggregated runtime status (includes schedules/session/subagent/cost metrics, recent command audit tail, and git upstream divergence)

### 7.3 Command Surface

- task lifecycle commands
- runtime configuration commands
- plugin/connector management commands

## 8. Runtime Lifecycle

Startup sequence:

1. initialize logger
2. load config
3. run startup preflight (config validity, SQLite writable path, executor binary availability)
4. open store and validate schema
5. run migration/backup checks
6. initialize plugin host + MCP bridge
7. initialize executor registry
8. start queue/scheduler
9. start gateway/channels

Shutdown sequence:

1. stop new intake
2. drain queue/scheduler/http with `runtime.shutdown.drain_timeout_sec`
3. close plugins/connectors
4. flush logs
5. close store

## 9. Observability and Operability

### 9.1 Logs

- structured logs with `session_id`, `task_id`, `stage`, `executor`, `plugin_id`
- dedicated audit streams for command and capability invocations, and status snapshot tail aggregation for quick diagnostics

### 9.2 Metrics

- queue depth, configured workers, in-flight workers, and last shutdown drain report
- schedule status/kind/delivery-mode counters, active overdue windows, and next active run marker
- session and sub-agent activity windows (`total/active/by_channel`) from orchestrator execution logs
- execution latency/error rate plus model usage estimates (`chars/tokens/cost`) with optional pricing config
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
- operators can refresh statuses, cancel `pending` requests, and retry `timeout/canceled` requests without leaving the browser

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
  core/
    interaction/
    gateway/
    orchestrator/
    runtime/
    queue/
    scheduler/
  executors/
    codex/
    claude_code/
    <future>
  extensions/
    skills/
      builtins/
      plugins/
    mcp/
      bridge/
      connectors/
  store/
    sqlite/
    migrations/
  observability/
    logging/
    metrics/
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

## Feature Roadmap

See `docs/features.md` for implemented capability matrix and active queue.
See `features.md` for the OpenClaw-aligned backlog of unshipped requirements.

Documentation consistency rule:

- Any PR that updates `docs/features.md` must also update `README.md` and `ARCHITECTURE.md` in the same change set.
- Use `make docs-sync-check` before opening a PR. The check compares `origin/master...HEAD` and fails when only `docs/features.md` changed.
