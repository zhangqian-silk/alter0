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
- [x] Runtime cost hotspot telemetry (`cost.session_hotspots`) with compaction pressure visibility (`prompt_output_ratio`)
- [x] Runtime cost threshold guidance (`cost.threshold_guidance`) for p90-based share/ratio tuning with drift hints, including workload-tier recommendations (`workload_tiers`)
- [x] Weekly cost threshold history archive with alert hit-rate regression (`make cost-threshold-history` -> `output/cost/threshold-history-latest.json` + `output/cost/threshold-history/<ISO-week>/`)
- [x] Threshold reconcile coordinator with bounded proposal/apply workflow (`make cost-threshold-reconcile` -> `output/cost/threshold-reconcile-latest.json` + optional `--apply`)
- [x] End-to-end gateway tracing + runtime alerts (`output/trace` + `/status` alerts for queue backlog/retry storm/channel disconnect/executor availability + session cost hotspot/compaction pressure)
- [x] Runtime risk watchlist snapshot (`config/risk-watchlist.json`) with stale/overdue policy/supply-chain alerts
- [x] Risk execution benchmark gate (`make risk-benchmark`) with JSON report output (`output/risk/benchmark-latest.json`) and drift triage runbook (`docs/runbooks/risk-drift-triage.md`), including threshold-history freshness checks
- [x] Scenario benchmark matrix + competitor tracking gate extension (`config/scenario-benchmark-matrix.json`, `config/competitor-tracking.json`, `scripts/update-competitor-tracking.sh`)
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
- [x] Release gates (`make release-gate`) with config boundary (`make check-config-boundary`) + service boundary (`make check-service-boundary`) + config parameter governance (`make config-governance`) + risk benchmark (`make risk-benchmark`) + cost threshold history/reconcile (`make cost-threshold-history`, `make cost-threshold-reconcile`) + test stability (`make test-stability`, Windows compile check) + rollback drill + docs/deploy checks
- [x] Config parameter governance audit for agents/bindings/session/tools (`make config-governance` -> `output/config/governance-latest.json`)
- [x] Service-layer schedule facade (`app/core/service/schedule`) used by runtime/HTTP integration boundaries
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
4. [x] N15 Service modularization (schedule domain service facade + boundary gate: `make check-service-boundary`)

### P2 (Risk Hardening)

1. [x] N16 External policy/supply-chain watchlist automation (`config/risk-watchlist.json` + runtime stale/overdue alerts)
2. [x] N17 Risk execution benchmark + drift triage runbook (`make risk-benchmark` + release-gate integration)
3. [x] N18 Scenario benchmark matrix + monthly competitor tracking automation (`config/scenario-benchmark-matrix.json` + `config/competitor-tracking.json` + `make competitor-tracking-refresh`)

### P3 (Governance Deep Dive)

1. [x] N19 Parameter-level config governance for agents/bindings/session/tools (`make config-governance` + release-gate integration)
2. [x] N20 Monthly governance cadence automation (snapshot archive + drift reminder)

### P4 (Runtime Cost Governance)

1. [x] N21 Session cost hotspot and compaction pressure alerts (`cost.session_hotspots` + `alerts.session_{cost_hotspot,compaction_pressure}`)
2. [x] N22 Session cost threshold guidance (`cost.threshold_guidance` with p90 recommendations and drift deltas)
3. [x] N23 Weekly threshold-history regression automation (`make cost-threshold-history` + risk benchmark linkage)
4. [x] N24 Workload-tier threshold guidance (`cost.threshold_guidance.workload_tiers` by token-volume buckets)
5. [x] N25 Threshold reconcile coordinator (`make cost-threshold-reconcile` generates bounded proposals with optional `--apply` config write-back)

Queue status: N25 merged; runtime cost governance backlog has no open blocking item.

## 4) Change Rule

When a feature state changes, update this file in the same PR:

- Keep capability matrix and active queue in sync
- Update item status ([ ], [~], [x])
- Keep non-goals explicit
- Sync roadmap narrative in `features.md`
- Run `make docs-sync-check BASE=origin/master` when a PR changes `docs/features.md`
