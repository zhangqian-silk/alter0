# Risk Drift Triage Runbook

## Scope

This runbook handles drift events from `provider_policy` and `supply_chain` watchlist categories,
and freshness/coverage drift from scenario and competitor tracking snapshots.
Use it with `make risk-benchmark` or `./scripts/check-risk-benchmark.sh`.

## Detection

1. Run `make competitor-tracking-refresh` (optional but recommended before monthly review).
2. Run `make risk-benchmark`.
3. Read `output/risk/benchmark-latest.json`.
4. Confirm whether `gate.passed` is `false`, any `drifts` entry exists, or scenario/competitor sections report failures.

## Classification

Classify each drift item by `drift_level` in the benchmark report:

- `critical`: high/critical severity, or overdue for 72h+
- `high`: medium+ severity, or overdue for 24h+
- `medium`: low urgency overdue drift

Classify scenario/competitor failures by impact:

- `scenario_matrix` missing required workloads or stale benchmark data -> release-blocking
- `competitor_tracking` stale project records or missing feature changes -> release-blocking
- non-critical metadata warnings (template/comment gaps) -> warning-only, patch in same week

## Notification

Notify by category and severity:

- `provider_policy` -> `provider-oncall`
- `supply_chain` -> `security-oncall`
- `scenario_matrix` / `competitor_tracking` -> `runtime-oncall`
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

### Scenario Benchmark Drift

1. Refresh benchmark inputs/results in `config/scenario-benchmark-matrix.json` for all required workloads.
2. Re-run `make risk-benchmark` and confirm `scenario_matrix.summary.status=pass`.
3. If latency/cost drift is high or critical, open an optimization task in the same milestone.

### Competitor Tracking Drift

1. Run `make competitor-tracking-refresh` (prefer `GH_TOKEN`/`GITHUB_TOKEN`) and verify API fetch results.
2. Add at least one `feature_changes` entry per tracked project for the monthly window.
3. Re-run `make risk-benchmark` and confirm `competitor_tracking.summary.status=pass`.

## Postmortem

1. Add timeline, impact, and fix summary to incident notes.
2. If new failure mode is found, append benchmark rules in `app/core/riskbench`.
3. If process changed, update this runbook in the same PR.
