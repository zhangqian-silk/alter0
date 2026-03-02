# OpenClaw Alignment Checklist

This checklist tracks Alter0 parity against the OpenClaw gateway baseline by delivery version.

## 2026.03 (current)

| Area | Capability | Status | Evidence |
| --- | --- | --- | --- |
| Gateway | Multi-channel adapters (CLI/HTTP + Telegram/Slack) | aligned | `app/core/interaction/*`, `go test ./app/core/interaction/...` |
| Agent orchestration | Multi-agent registry + sub-agent run/session | aligned | `app/core/interaction/gateway`, `app/core/interaction/http` tests |
| Scheduling | One-shot `at` + recurring `cron` with direct delivery | aligned | `app/core/schedule/service_test.go` |
| Tool runtime | Normalized protocol + policy gates + audit | aligned | `app/core/tools/runtime_test.go` |
| Memory safety | Trusted-channel memory isolation (`memory_search/memory_get`) | aligned | `app/core/tools/runtime_test.go` |
| Runtime ops | Trace/alerts + queue/scheduler/subagent metrics + channel degradation snapshot/fallback guidance (`channel_degradation`) + configurable channel degradation thresholds (`runtime.observability.channel_degradation` default + per-channel overrides with suppressed-noise accounting) + channel chaos drill regression gate (`make channel-chaos-drill`) + session cost hotspot/compaction pressure signals + threshold guidance (`cost.threshold_guidance`, incl. `workload_tiers`) + weekly threshold-history regression archive + bounded threshold reconcile/apply coordinator + reconcile cadence archive/trend telemetry | aligned | `app/core/runtime/status_test.go`, `app/configs/config_test.go`, `app/core/channelchaos/drill_test.go`, `app/cmd/channel-chaos-drill/main.go`, `app/cmd/cost-threshold-history/main.go`, `app/cmd/cost-threshold-reconcile/main.go` |
| Release safety | Integration matrix + risk benchmark + channel chaos drill + cost threshold history/reconcile cadence + rollback drill + docs sync gates | aligned | `scripts/run-integration-matrix.sh`, `scripts/check-risk-benchmark.sh`, `scripts/check-channel-chaos-drill.sh`, `scripts/check-cost-threshold-history.sh`, `scripts/check-cost-threshold-reconcile.sh`, `scripts/check-config-governance.sh`, `config/scenario-benchmark-matrix.json`, `config/competitor-tracking.json`, `config/channel-chaos-matrix.json`, `scripts/update-competitor-tracking.sh`, `scripts/check-release-gates.sh` |

## Gap Policy

- Any non-aligned item must include an owner, target milestone, and blocking dependency in this file.
- `2026.03` is release-ready only when every row is `aligned` and `make release-gate` is green.
