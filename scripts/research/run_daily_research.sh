#!/usr/bin/env bash
set -euo pipefail

ROOT="/root/.openclaw/workspace/alter0"
cd "$ROOT"

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"
export TZ="Asia/Shanghai"

PROMPT_FILE="$ROOT/scripts/research/daily_research_agent_prompt.md"
if [ ! -f "$PROMPT_FILE" ]; then
  echo "missing prompt file: $PROMPT_FILE" >&2
  exit 1
fi

PROMPT_CONTENT=$(cat "$PROMPT_FILE")
DATE_TAG=$(date +%F)

openclaw agent \
  --message "[Scheduled Daily Research ${DATE_TAG}] ${PROMPT_CONTENT}" \
  --thinking high \
  --timeout 1800 \
  --json
