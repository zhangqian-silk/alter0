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
  go test ./app/core/schedule ./app/core/tools

run "fault injection matrix (disconnect/retry/queue/db-lock)" \
  go test ./app/core/interaction/gateway ./app/core/queue ./app/core/orchestrator/db -run 'TestGatewayTracesChannelDisconnect|TestQueueRecoversAfterJobFailure|TestNewSQLiteDBReturnsLockErrorWhenSchemaLocked'

run "restart recovery matrix" \
  go test ./app/core/schedule -run TestServiceRecoversDueAtJobAfterRestart

echo "integration matrix passed"
