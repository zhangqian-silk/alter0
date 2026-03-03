#!/usr/bin/env bash
set -euo pipefail

ROOT="/root/.openclaw/workspace/alter0"
cd "$ROOT"

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"
export TZ="Asia/Shanghai"

python3 "$ROOT/scripts/research/daily_research_report.py"
