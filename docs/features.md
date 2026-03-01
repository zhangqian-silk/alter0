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
- [x] Deployment assets for systemd service + Docker image

### Interaction and Observability

- [x] CLI channel + slash commands
- [x] HTTP channel (`/api/message`, `/api/tasks`, `/health`, `/api/status`)
- [x] External chat channel adapters (Telegram polling + Slack events with outbound media envelope support)
- [x] Channel Adapter V2 envelope baseline (`Message.Envelope` with typed text/image/audio/file/reference parts)
- [x] Async task list API (`GET /api/tasks`) for recent request diagnostics
- [x] Web console async task panel (refresh/cancel/retry + task reuse)
- [x] Command permission audit JSONL
- [x] Runtime status snapshot command audit tail
- [x] Runtime status snapshot git upstream divergence (`upstream/ahead/behind`)
- [x] Executor stage JSONL logs

### Execution and Extensibility

- [x] Executor abstraction (`codex`, `claude_code`)
- [x] Executor capability registry in runtime status snapshot
- [x] Skill manager and built-in skills
- [x] Queue-driven async pipeline on core execution path
- [x] Queue status snapshot includes configured workers for runtime diagnostics
- [x] Queue status snapshot includes last shutdown drain report for timeout diagnostics

## 3) Active Priority Queue (OpenClaw Alignment)

Execution policy: complete one requirement end-to-end (`code -> test -> PR -> merge`) before picking next.

### P0 (Gateway Foundation)

1. [x] N1 Multi-channel gateway baseline with at least two external chat channels
2. [x] N2 Multi-agent registry and session isolation (`agentId/workspace/agentDir`)
3. [x] N3 Scheduler center for one-shot `at` + recurring `cron` with direct channel delivery
4. [ ] N6 Runtime status expansion (sessions/subagents/schedules/cost metrics)

### P1 (Orchestration Upgrade)

1. [ ] N2 Sub-agent run/session modes with result announce chain
2. [ ] N4 Tool protocol normalization (`web_search/web_fetch/browser/canvas/nodes/message/tts`) + policy gates
3. [ ] N5 Long-term memory retrieval and context-safety isolation by surface
4. [ ] N7 Security posture checks and high-risk operation guards

### P2 (Advanced Operations)

1. [ ] N4 Deep node/browser/canvas toolchain integration
2. [ ] N6 End-to-end tracing and alerting hardening
3. [ ] N8 Full integration matrix and release gates for rollback-safe delivery

Queue status: historical P0/P1 items are merged and retained in Capability Matrix; active queue now tracks OpenClaw-aligned gateway requirements only.

## 4) Change Rule

When a feature state changes, update this file in the same PR:

- Keep capability matrix and active queue in sync
- Update item status ([ ], [~], [x])
- Keep non-goals explicit
- Sync roadmap narrative in `features.md`
- Run `make docs-sync-check BASE=origin/master` when a PR changes `docs/features.md`
