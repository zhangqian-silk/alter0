# Risk Drift Triage Runbook

## Scope

This runbook handles drift events from `provider_policy` and `supply_chain` watchlist categories.
Use it with `make risk-benchmark` or `./scripts/check-risk-benchmark.sh`.

## Detection

1. Run `make risk-benchmark`.
2. Read `output/risk/benchmark-latest.json`.
3. Confirm whether `gate.passed` is `false` or any `drifts` entry exists.

## Classification

Classify each drift item by `drift_level` in the benchmark report:

- `critical`: high/critical severity, or overdue for 72h+
- `high`: medium+ severity, or overdue for 24h+
- `medium`: low urgency overdue drift

## Notification

Notify by category and severity:

- `provider_policy` -> `provider-oncall`
- `supply_chain` -> `security-oncall`
- Unknown category -> `runtime-oncall`

Escalation windows:

- `critical`: notify in 15 minutes, create incident ticket immediately
- `high`: notify in 1 hour, assign mitigation owner in same shift
- `medium`: notify within 24 hours, include in daily ops review

## Recovery

### Provider Policy Drift

1. Validate recent provider/channel policy changes (auth scope, quota, account status).
2. Apply fallback route or temporary executor/channel policy override.
3. Re-run smoke tests for gateway message ingress + agent execution.
4. Update `config/risk-watchlist.json` (`last_checked_at`, `next_review_at`, notes).

### Supply Chain Drift

1. Identify changed plugin/skill source or version mismatch.
2. Freeze untrusted source, pin approved version, and re-run integration matrix.
3. Run `make release-gate` after mitigation to ensure baseline is clean.
4. Update `config/risk-watchlist.json` (`status`, `notes`, review timestamp).

## Postmortem

1. Add timeline, impact, and fix summary to incident notes.
2. If new failure mode is found, append benchmark rules in `app/core/riskbench`.
3. If process changed, update this runbook in the same PR.
