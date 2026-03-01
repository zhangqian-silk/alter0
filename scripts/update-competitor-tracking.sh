#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TRACKING_PATH="${TRACKING_PATH:-config/competitor-tracking.json}"
GH_API_BASE="${GH_API_BASE:-https://api.github.com}"
STRICT_FETCH="${STRICT_FETCH:-false}"

cd "$ROOT_DIR"

python3 - "$TRACKING_PATH" "$GH_API_BASE" "$STRICT_FETCH" <<'PY'
import datetime as dt
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

tracking_path = sys.argv[1]
api_base = sys.argv[2].rstrip("/")
strict_fetch = sys.argv[3].strip().lower() == "true"

with open(tracking_path, "r", encoding="utf-8") as fp:
    doc = json.load(fp)

now = dt.datetime.now(dt.timezone.utc)
since_30d = now - dt.timedelta(days=30)
since_90d = now - dt.timedelta(days=90)

headers = {
    "Accept": "application/vnd.github+json",
    "User-Agent": "alter0-competitor-tracking/1.0",
}

token = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")
if token:
    headers["Authorization"] = f"Bearer {token}"


def parse_time(value):
    if not value:
        return None
    try:
        return dt.datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(dt.timezone.utc)
    except ValueError:
        return None


def to_rfc3339(value):
    if not value:
        return ""
    return value.astimezone(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def build_url(path, query=None):
    url = f"{api_base}{path}"
    if query:
        url = f"{url}?{urllib.parse.urlencode(query)}"
    return url


def request_json_url(url):
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req, timeout=20) as resp:
        payload = json.loads(resp.read().decode("utf-8"))
        return payload, resp.headers.get("Link", "")


def request_json(path, query=None):
    payload, _ = request_json_url(build_url(path, query))
    return payload


def parse_next_link(link_header):
    if not link_header:
        return ""
    match = re.search(r'<([^>]+)>;\s*rel="next"', link_header)
    if not match:
        return ""
    return match.group(1)


def fetch_paginated(path, query=None):
    results = []
    next_url = build_url(path, query)
    while next_url:
        payload, link_header = request_json_url(next_url)
        if not isinstance(payload, list):
            break
        results.extend(payload)
        next_url = parse_next_link(link_header)
    return results


warnings = []
updated_projects = 0

for project in doc.get("projects", []):
    repo = (project.get("repo") or "").strip()
    if not repo:
        raise SystemExit("project repo is required in competitor-tracking.json")

    try:
        repo_info = request_json(f"/repos/{repo}")
        commits = fetch_paginated(
            f"/repos/{repo}/commits",
            {
                "since": to_rfc3339(since_30d),
                "per_page": 100,
            },
        )
        releases = fetch_paginated(
            f"/repos/{repo}/releases",
            {
                "per_page": 100,
            },
        )
    except urllib.error.HTTPError as exc:
        message = exc.read().decode("utf-8", "ignore")
        detail = f"github api error for {repo}: {exc.code} {message}"
        if strict_fetch:
            raise SystemExit(detail) from exc
        warnings.append(detail)
        continue

    release_count = 0
    last_release = None
    for item in releases:
        published = parse_time(item.get("published_at"))
        if not published:
            continue
        if not last_release or published > last_release:
            last_release = published
        if published >= since_90d:
            release_count += 1

    last_commit = parse_time(repo_info.get("pushed_at"))
    project["stars"] = int(repo_info.get("stargazers_count") or 0)
    project.setdefault("activity", {})
    project.setdefault("release", {})
    project["activity"]["last_commit_at"] = to_rfc3339(last_commit)
    project["activity"]["commits_30d"] = len(commits)
    project["release"]["last_release_at"] = to_rfc3339(last_release)
    project["release"]["releases_90d"] = release_count
    project["last_checked_at"] = to_rfc3339(now)
    updated_projects += 1

if updated_projects > 0:
    doc["updated_at"] = to_rfc3339(now)

with open(tracking_path, "w", encoding="utf-8") as fp:
    json.dump(doc, fp, indent=2)
    fp.write("\n")

print(f"updated competitor tracking snapshot: {tracking_path} (projects refreshed: {updated_projects})")
if warnings:
    print("warnings:")
    for item in warnings:
        print(f" - {item}")
PY
