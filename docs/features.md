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
- [x] Runtime maintenance jobs (task memory prune)
- [x] Unified runtime status snapshot (gateway/scheduler/task/queue/git)
- [x] Startup preflight checks (SQLite writable, executor installed, config validation)
- [x] Configurable graceful shutdown drain timeout (queue/scheduler/http unified)
- [x] Local backup/restore scripts for DB + config
- [x] Deployment assets for systemd service + Docker image

### Interaction and Observability

- [x] CLI channel + slash commands
- [x] HTTP channel (`/api/message`, `/health`, `/api/status`)
- [x] Command permission audit JSONL
- [x] Runtime status snapshot command audit tail
- [x] Executor stage JSONL logs

### Execution and Extensibility

- [x] Executor abstraction (`codex`, `claude_code`)
- [x] Executor capability registry in runtime status snapshot
- [x] Skill manager and built-in skills
- [x] Queue-driven async pipeline on core execution path
- [x] Queue status snapshot includes configured workers for runtime diagnostics
- [x] Queue status snapshot includes last shutdown drain report for timeout diagnostics

## 3) Active Priority Queue

Execution policy: complete one requirement end-to-end (`code -> test -> PR -> merge`) before picking next.

### P0

1. [x] README/ARCHITECTURE sync after each feature merge
2. [x] Startup preflight checks (SQLite writable, executor installed, config validation)
3. [x] Graceful shutdown drain timeout configurable and shared by queue/scheduler/http
4. [x] Deployment assets: systemd service template + Dockerfile runtime packaging

### P1

1. [x] Runtime status snapshot includes executor capability registry (name/command/installed)
2. [x] Runtime status snapshot adds recent command audit tail for quick diagnostics
3. [x] Queue stats expose in-flight worker count for `/status` and `/api/status` diagnostics
4. [x] Queue stats expose configured worker count for `/status` and `/api/status` diagnostics
5. [x] Queue stats expose last shutdown drain report for `/status` and `/api/status` diagnostics

Queue status: P0/P1 active queue is clear (all listed items merged); docs sync guard is enforced via `scripts/check-doc-sync.sh`, and deployment asset guard via `scripts/check-deploy-assets.sh`.

## 4) Change Rule

When a feature is merged, update this file in the same PR:

- Move item status ([ ], [~], [x])
- Update active queue
- Keep non-goals explicit
- Run `make docs-sync-check BASE=origin/master` when a PR changes `docs/features.md`
