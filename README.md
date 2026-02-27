# Alter0

Alter0 is a multi-channel task orchestrator that executes user requests via a configured executor (`claude_code` or `codex`).

## Runtime Flow

`Channel -> Gateway -> Orchestrator(Task Router + Task Store) -> Executor -> Orchestrator -> Channel`

- Channels currently included: CLI + HTTP/Web
- Conversation state is task-centric (not channel-centric)
- Task routing and task closing are model-assisted
- HTTP returns synchronous responses with `task_id`, `decision`, and `closed`

## Commands

- `/help`
- `/config`
- `/config get [key]`
- `/config set <key> <value>`
- `/executor [name]`
- `/task list [open|closed|all]`
- `/task current`
- `/task use <task_id>`
- `/task new [title]`
- `/task close [task_id]`

## Config

The runtime config file is [`config/config.json`](./config/config.json):

```json
{
  "agent": {"name": "Alter0"},
  "executor": {"name": "claude_code"},
  "task": {
    "routing_timeout_sec": 15,
    "close_timeout_sec": 10,
    "generation_timeout_sec": 120,
    "routing_confidence_threshold": 0.55,
    "close_confidence_threshold": 0.7,
    "cli_user_id": "local_user",
    "open_task_candidate_limit": 8
  }
}
```

## Run

```bash
go run .
```

Then open:

- Web UI: `http://localhost:8080/`
- API: `POST http://localhost:8080/api/message`
