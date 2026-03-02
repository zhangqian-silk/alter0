#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

run() {
  local label="$1"
  shift
  echo "==> $label"
  "$@"
}

cd "$ROOT_DIR"

run "multi-channel gateway baseline" \
  go test ./app/core/interaction/gateway ./app/core/interaction/telegram ./app/core/interaction/slack

run "multi-agent + subagent matrix" \
  go test ./app/core/interaction/http -run 'TestSubagentRunModeCompletesAndAnnounces|TestSubagentSessionModeSupportsTurns'

run "scheduler + tool protocol matrix" \
  go test ./app/core/orchestrator/schedule ./app/core/service/schedule ./app/core/runtime/tools ./app/core/runtime -run 'TestSnapshotIncludesExpandedScheduleMetrics|TestSnapshotIncludesToolProtocolSnapshot'

run "fault injection matrix (disconnect/retry/queue/db-lock)" \
  go test ./app/core/interaction/gateway -run 'TestGatewayTracesChannelDisconnect|TestGatewayDispatchWithQueueRetries'

run "channel chaos drill matrix" \
  go test ./app/core/channelchaos -run 'TestRunMatrixPassesScenario|TestRunMatrixFailsWhenFallbackMissing|TestBuildCandidatesFromTraceGeneratesFallbackScenario|TestBuildCandidatesFromTraceHonorsMinErrorsAndLimit'

run "restart recovery matrix" \
  go test ./app/core/orchestrator/task -run 'TestTaskMemorySnapshotRestoreIsAtomic|TestPruneTaskMemoryByClosedAt'

echo "integration matrix passed"
