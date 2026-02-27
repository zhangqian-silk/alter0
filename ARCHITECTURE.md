# Alter0 Architecture

## Core Design

Alter0 uses a task-centric orchestration model:

1. Channel receives a message (`cli`, `http`)
2. Gateway forwards message to Agent
3. Agent (Orchestrator):
   - resolves target task (forced task > explicit task_id > model routing > new task)
   - persists messages in SQLite
   - delegates generation to executor
   - decides whether to close task
4. Gateway sends structured response back through originating channel
5. Main runtime wraps Gateway with retry/backoff restart loop for self-healing

## Components

- `app/core/interaction/*`
  - channel adapters (`cli`, `http`)
  - `gateway` for channel registration and message dispatch
- `app/core/orchestrator/agent`
  - central orchestration flow
- `app/core/orchestrator/command`
  - slash command handling with permission/audit logs
- `app/core/orchestrator/task`
  - task store, routing, close decision, and task memory snapshot persistence
- `app/core/agent/executor.go`
  - executor process invocation (`claude_code` or `codex`)
- `app/core/orchestrator/db/sqlite.go`
  - schema/version bootstrap
- `app/core/scheduler/scheduler.go`
  - interval-based scheduler lifecycle (`Register`, `Start`, `Stop`)
- `app/core/queue/queue.go`
  - in-memory worker queue with retry, per-attempt timeout, and cancellation-aware shutdown
- `app/configs/config.go`
  - runtime config manager

## Data Model

SQLite schema:

- `tasks`: task lifecycle and ownership
- `messages`: task-scoped conversation messages
- `user_state`: one-shot forced task overrides (`/task use`)
- `task_memory`: compact per-task memory snapshot for prompt reuse
- `schema_meta`: schema version marker

## API

`POST /api/message`

Request:

```json
{"content":"...", "user_id":"...", "task_id":"optional"}
```

Response:

```json
{"task_id":"...", "response":"...", "closed":false, "decision":"existing|new"}
```
