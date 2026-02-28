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
- [x] Unified runtime status snapshot (gateway/scheduler/task/git)
- [x] Startup preflight checks (SQLite writable, executor installed, config validation)
- [x] Local backup/restore scripts for DB + config

### Interaction and Observability

- [x] CLI channel + slash commands
- [x] HTTP channel (`/api/message`, `/health`, `/api/status`)
- [x] Command permission audit JSONL
- [x] Executor stage JSONL logs

### Execution and Extensibility

- [x] Executor abstraction (`codex`, `claude_code`)
- [x] Skill manager and built-in skills
- [x] Queue-driven async pipeline on core execution path

## 3) Active Priority Queue

Execution policy: complete one requirement end-to-end (`code -> test -> PR -> merge`) before picking next.

### P0

1. [x] README/ARCHITECTURE sync after each feature merge
2. [x] Startup preflight checks (SQLite writable, executor installed, config validation)

### P1 (Do next)

1. [x] Graceful shutdown drain policy for in-flight queue tasks with timeout metrics
2. [ ] Regression tests for startup preflight failure paths

Queue status: P1 graceful shutdown drain policy merged; preflight regression tests are next.

## 4) Change Rule

When a feature is merged, update this file in the same PR:

- Move item status ([ ], [~], [x])
- Update active queue
- Keep non-goals explicit
