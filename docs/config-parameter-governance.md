# Config Parameter Governance (N19)

This document defines the parameter-level governance baseline for Alter0 runtime config.

- Audit command: `make config-governance`
- Output report: `output/config/governance-latest.json`
- Gate integration: `make release-gate` (via `scripts/check-config-governance.sh`)

## Domain Coverage

The audit follows four domains aligned with the OpenClaw report chapter 5.2 guidance.

### 1) Agents

| Path | Default | Risk | Gate rule |
| --- | --- | --- | --- |
| `agent.default_id` | `default` | high | Must reference an id present in `agent.registry`. |
| `agent.registry` | one default agent entry | high | Must not be empty or contain duplicate agent ids. |
| `executor.name` | `claude_code` | medium | Warn when executor is outside curated baseline (`codex/claude_code/opencode`). |

### 2) Bindings

| Path | Default | Risk | Gate rule |
| --- | --- | --- | --- |
| `channels.telegram.enabled` | `false` | high | When enabled, `bot_token` is required; missing `default_chat_id` is warning. |
| `channels.slack.enabled` | `false` | high | When enabled, both `bot_token` and `app_id` are required. |

### 3) Session

| Path | Default | Risk | Gate rule |
| --- | --- | --- | --- |
| `task.routing_confidence_threshold` | `0.55` | high | Fail if not in `(0,1)`; warn when outside `[0.45,0.85]`. |
| `task.close_confidence_threshold` | `0.70` | medium | Fail if not in `(0,1)`; warn when outside `[0.55,0.90]`. |
| `task.generation_timeout_sec` | `120` | high | Fail if `<30` or `>1800`; warn if `>300`. |
| `runtime.queue.workers` | `2` | high | Fail if `<=0` or `>32`; warn if `>8`. |
| `runtime.queue.max_retries` | `1` | medium | Fail if `<0` or `>10`; warn if `>3`. |
| `runtime.shutdown.drain_timeout_sec` | `5` | high | Fail if `<3` or `>180`; warn if `>60`. |

### 4) Tools

| Path | Default | Risk | Gate rule |
| --- | --- | --- | --- |
| `security.admin_user_ids` | `local_user` | high | Must be non-empty; wildcard `*` is forbidden. |
| `security.tools.require_confirm` | `browser,canvas,nodes,message` | critical | Must contain all four high-risk tools. |
| `security.tools.global_allow` + `security.tools.global_deny` | empty | high | Same tool cannot appear in both lists. |
| `security.memory.trusted_channels` | `cli,http` | high | Must be non-empty; missing `memory.md` in restricted paths is warning. |

## Default Governance Policy

1. If a parameter is omitted from `config/config.json`, Alter0 applies default values and marks the field as `explicit=false` in the audit report.
2. High-risk and critical parameters should be explicitly configured in production-like environments.
3. Any fail-level check blocks `make config-governance` and therefore blocks `make release-gate`.
4. Warning-level checks do not block merge, but should be resolved in the same milestone when feasible.

## Operational Workflow

1. Update `config/config.json`.
2. Run `make config-governance`.
3. Review `output/config/governance-latest.json`.
4. If fail-level items exist, fix config before running `make release-gate`.
