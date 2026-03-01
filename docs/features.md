# Alter0 Features

## 1) Product Positioning

Alter0 is a single-user, self-hosted task orchestration kernel.

- Default mode: local-first, single instance
- Priority: reliability, maintainability, observability
- Non-goals: multi-tenant SaaS controls (OAuth/RBAC/billing)

## 2) Capability Matrix

### Task Orchestration

- [x] Task as first-class entity (`create`, `route`, `close`, `force-use`)
- [x] LLM router + closer decision chain
- [x] Task memory snapshot (`upsert/get/clear/export/restore`)

### Runtime and Recovery

- [x] Gateway crash backoff restart loop
- [x] Dynamic scheduler register/unregister + run-on-start
- [x] Persistent schedule center for one-shot `at` + recurring `cron` with direct channel delivery
- [x] Runtime maintenance jobs (task memory prune)
- [x] Unified runtime status snapshot (gateway/scheduler/task/queue/git)
- [x] Startup preflight checks (SQLite writable, executor installed, config validation)
- [x] Multi-agent registry and session isolation (`agentId/workspace/agentDir` with default fallback)
- [x] Configurable graceful shutdown drain timeout (queue/scheduler/http unified)
- [x] Local backup/restore scripts for DB + config
- [x] Backup/restore rollback drill automation (`make rollback-drill`)
- [x] Deployment assets for systemd service + Docker image

### Interaction and Observability

- [x] CLI channel + slash commands
- [x] HTTP channel (`/api/message`, `/api/tasks`, `/health`, `/api/status`)
- [x] External chat channel adapters (Telegram polling + Slack events with outbound media envelope support)
- [x] Channel Adapter V2 envelope baseline (`Message.Envelope` with typed text/image/audio/file/reference parts)
- [x] Async task list API (`GET /api/tasks`) for recent request diagnostics
- [x] Async task cancellation runtime semantics (`POST /api/tasks/{id}/cancel` supports pending/running interruption + canceled audit fields)
- [x] Web console async task panel (refresh/cancel/retry + task reuse)
- [x] Command permission audit JSONL
- [x] Runtime status snapshot command audit tail
- [x] Runtime status snapshot git upstream divergence (`upstream/ahead/behind`)
- [x] Runtime status expansion for sessions/subagents/schedules/cost metrics
- [x] End-to-end gateway tracing + runtime alerts (`output/trace` + `/status` alerts for queue backlog/retry storm/channel disconnect/executor availability)
- [x] Sub-agent run/session modes with result announce chain
- [x] Executor stage JSONL logs
- [x] Security posture checks with high-risk operation guardrails (`tools.security_posture` + baseline confirmation)

### Execution and Extensibility

- [x] Executor abstraction (`codex`, `claude_code`)
- [x] Executor capability registry in runtime status snapshot
- [x] Skill manager and built-in skills
- [x] Queue-driven async pipeline on core execution path
- [x] Queue status snapshot includes configured workers for runtime diagnostics
- [x] Queue status snapshot includes last shutdown drain report for timeout diagnostics
- [x] Tool protocol normalization (`web_search/web_fetch/browser/canvas/nodes/message/tts/memory_search/memory_get`) with policy gates and execution audit JSONL
- [x] Deep node/browser/canvas action schema exposure + structured argument validation (`tools.protocol.toolchain`)
- [x] Long-term memory retrieval protocol (`memory_search/memory_get`) with shared-surface safety isolation (`security.memory`)
- [x] Gateway integration matrix automation (`make integration-matrix`)
- [x] Release gates (`make release-gate`) with config boundary (`make check-config-boundary`) + test stability (`make test-stability`, Windows compile check) + rollback drill + docs/deploy checks
- [x] OpenClaw alignment checklist by release version (`docs/openclaw-alignment.md`)

## 3) Active Priority Queue (Post-Alignment)

Execution policy: complete one requirement end-to-end (`code -> test -> PR -> merge`) before picking next.

### P0 (Current Delivery)

1. [x] N9 Async task cancellation semantics hardening (pending/running interruption + canceled audit fields)
2. [x] N10 Storage migration safety upgrade (incremental migration + backup/rollback path)
3. [x] N14 Test stability hardening (`go test ./...` deterministic pass)

### P1 (Structural Refactor)

1. [x] N11 HTTP layer responsibility convergence (state sinks to service layer)
2. [x] N12 Scheduler boundary unification (single scheduling loop authority)
3. [x] N13 Config layer dependency decoupling (`configs` reverse dependency removed + `make check-config-boundary` guard)
4. [ ] N15 Service modularization (`core -> service -> infra` layering)

Queue status: N13 is merged in this round. Next item is `P1/N15`.

## 4) Change Rule

When a feature state changes, update this file in the same PR:

- Keep capability matrix and active queue in sync
- Update item status ([ ], [~], [x])
- Keep non-goals explicit
- Sync roadmap narrative in `features.md`
- Run `make docs-sync-check BASE=origin/master` when a PR changes `docs/features.md`
