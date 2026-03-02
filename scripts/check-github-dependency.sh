#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_PATH="${REPO_PATH:-$ROOT_DIR}"
OUTPUT_PATH="${OUTPUT_PATH:-$ROOT_DIR/output/delivery/github-dependency-latest.json}"
MAX_ATTEMPTS="${GITHUB_DEP_RETRIES:-4}"
RETRY_DELAY_SECONDS="${GITHUB_DEP_RETRY_DELAY:-5}"
CONNECT_TIMEOUT_SECONDS="${GITHUB_DEP_CONNECT_TIMEOUT:-20}"
ALLOW_NETWORK_FAILURE="${GITHUB_DEP_ALLOW_NETWORK_FAILURE:-0}"
REQUIRE_TOKEN="${GITHUB_DEP_REQUIRE_TOKEN:-0}"

if ! [[ "$MAX_ATTEMPTS" =~ ^[0-9]+$ ]] || [ "$MAX_ATTEMPTS" -lt 1 ]; then
  echo "invalid GITHUB_DEP_RETRIES: $MAX_ATTEMPTS" >&2
  exit 2
fi
if ! [[ "$RETRY_DELAY_SECONDS" =~ ^[0-9]+$ ]]; then
  echo "invalid GITHUB_DEP_RETRY_DELAY: $RETRY_DELAY_SECONDS" >&2
  exit 2
fi
if ! [[ "$CONNECT_TIMEOUT_SECONDS" =~ ^[0-9]+$ ]] || [ "$CONNECT_TIMEOUT_SECONDS" -lt 1 ]; then
  echo "invalid GITHUB_DEP_CONNECT_TIMEOUT: $CONNECT_TIMEOUT_SECONDS" >&2
  exit 2
fi

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  printf '%s' "$value"
}

status="ok"
ready=true
network_ok=false
attempts=0
failure_reason="none"
remote="origin"
remote_url=""

if ! remote_url="$(git -C "$REPO_PATH" remote get-url "$remote" 2>/dev/null)"; then
  status="missing_remote"
  ready=false
  failure_reason="missing_remote"
fi

if [ "$status" = "ok" ]; then
  while [ "$attempts" -lt "$MAX_ATTEMPTS" ]; do
    attempts=$((attempts + 1))
    if timeout "${CONNECT_TIMEOUT_SECONDS}s" git -C "$REPO_PATH" ls-remote --heads "$remote" >/dev/null 2>&1; then
      network_ok=true
      break
    fi
    failure_reason="network_unreachable"
    if [ "$attempts" -lt "$MAX_ATTEMPTS" ]; then
      sleep "$RETRY_DELAY_SECONDS"
    fi
  done

  if [ "$network_ok" != true ]; then
    status="network_unreachable"
    ready=false
  fi
fi

token_present=false
if [ -n "${GH_TOKEN:-}" ] || [ -n "${GITHUB_TOKEN:-}" ]; then
  token_present=true
fi

if [ "$REQUIRE_TOKEN" = "1" ] && [ "$token_present" != true ]; then
  status="token_missing"
  ready=false
  failure_reason="token_missing"
fi

if [ "$ready" = true ]; then
  failure_reason="none"
fi

if [ "$ready" = true ]; then
  recommendation="github dependency checks passed"
elif [ "$status" = "missing_remote" ]; then
  recommendation="configure git remote origin before running delivery pipeline"
elif [ "$status" = "token_missing" ]; then
  recommendation="set GH_TOKEN or GITHUB_TOKEN before creating PRs"
else
  recommendation="retry after network recovery or run with GITHUB_DEP_ALLOW_NETWORK_FAILURE=1 for degraded mode"
fi

mkdir -p "$(dirname "$OUTPUT_PATH")"

generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

cat >"$OUTPUT_PATH" <<EOF
{
  "generated_at": "$(json_escape "$generated_at")",
  "repo_path": "$(json_escape "$REPO_PATH")",
  "remote": "$(json_escape "$remote")",
  "remote_url": "$(json_escape "$remote_url")",
  "status": "$(json_escape "$status")",
  "ready": $ready,
  "attempts": $attempts,
  "network_ok": $network_ok,
  "token_present": $token_present,
  "token_required": $( [ "$REQUIRE_TOKEN" = "1" ] && echo true || echo false ),
  "failure_reason": "$(json_escape "$failure_reason")",
  "retry_policy": {
    "max_attempts": $MAX_ATTEMPTS,
    "retry_delay_seconds": $RETRY_DELAY_SECONDS,
    "connect_timeout_seconds": $CONNECT_TIMEOUT_SECONDS
  },
  "recommendation": "$(json_escape "$recommendation")"
}
EOF

if [ "$ready" = true ]; then
  echo "github dependency check passed; report=$OUTPUT_PATH"
  exit 0
fi

if [ "$ALLOW_NETWORK_FAILURE" = "1" ] && [ "$status" = "network_unreachable" ]; then
  echo "github dependency check degraded: network unavailable but allowed; report=$OUTPUT_PATH" >&2
  exit 0
fi

echo "github dependency check failed: status=$status report=$OUTPUT_PATH" >&2
exit 1
