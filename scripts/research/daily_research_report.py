#!/usr/bin/env python3
import base64
import datetime as dt
import json
import os
import re
import subprocess
import sys
from collections import Counter
from pathlib import Path
from urllib.parse import quote_plus

ROOT = Path(__file__).resolve().parents[2]
DOCS_DIR = ROOT / "docs"
RESEARCH_DIR = DOCS_DIR / "research"
REQ_FILE = DOCS_DIR / "requirements.md"
DOCS_INDEX = DOCS_DIR / "README.md"

REPO_CANDIDATES = [
    "openai/codex",
    "anthropics/claude-code",
    "Aider-AI/aider",
    "langchain-ai/langgraph",
    "openai/openai-agents-python",
    "modelcontextprotocol/modelcontextprotocol",
    "openclaw/openclaw",
]

AI_HOTSPOT_QUERIES = [
    ("agent", "ai agent framework"),
    ("coding", "ai coding assistant"),
    ("mcp", "model context protocol mcp"),
    ("rag", "retrieval augmented generation ai"),
    ("infra", "llm inference serving"),
    ("multimodal", "multimodal ai vision audio"),
]

SH_TZ = dt.timezone(dt.timedelta(hours=8))


def run(cmd, check=True):
    proc = subprocess.run(cmd, cwd=ROOT, text=True, capture_output=True)
    if check and proc.returncode != 0:
        raise RuntimeError(f"cmd failed: {' '.join(cmd)}\n{proc.stderr.strip()}")
    return proc.stdout.strip()


def gh_api(path, method="GET", fields=None, check=True):
    cmd = ["gh", "api", path]
    if method != "GET":
        cmd += ["-X", method]
    if fields:
        for key, value in fields.items():
            cmd += ["-f", f"{key}={value}"]
    out = run(cmd, check=check)
    if not out:
        return None
    return json.loads(out)


def parse_owner_repo():
    env_repo = os.getenv("ALTER0_REPO")
    if env_repo and "/" in env_repo:
        return env_repo
    url = run(["git", "remote", "get-url", "origin"]) 
    m = re.search(r"[:/]([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+?)(?:\.git)?$", url)
    if not m:
        return "zhangqian-silk/alter0"
    return m.group(1)


def fetch_repo_stats():
    rows = []
    for repo in REPO_CANDIDATES:
        data = gh_api(f"repos/{repo}")
        rows.append(
            {
                "repo": repo,
                "stars": data.get("stargazers_count", 0),
                "updated_at": data.get("updated_at", ""),
                "pushed_at": data.get("pushed_at", ""),
                "open_issues": data.get("open_issues_count", 0),
            }
        )
    return rows


def fetch_ai_hotspots():
    cutoff = (dt.datetime.now(SH_TZ) - dt.timedelta(days=14)).strftime("%Y-%m-%d")
    repos = {}

    for signal, phrase in AI_HOTSPOT_QUERIES:
        query = f"{phrase} stars:>150 pushed:>={cutoff}"
        path = f"search/repositories?q={quote_plus(query)}&sort=updated&order=desc&per_page=8"
        payload = gh_api(path, check=False) or {}
        for item in payload.get("items", []):
            full_name = item.get("full_name")
            if not full_name:
                continue
            row = repos.get(full_name)
            if not row:
                row = {
                    "repo": full_name,
                    "stars": item.get("stargazers_count", 0),
                    "updated_at": item.get("updated_at", ""),
                    "pushed_at": item.get("pushed_at", ""),
                    "signals": set(),
                }
                repos[full_name] = row
            row["signals"].add(signal)
            if item.get("updated_at", "") > row["updated_at"]:
                row["updated_at"] = item.get("updated_at", "")
            if item.get("pushed_at", "") > row["pushed_at"]:
                row["pushed_at"] = item.get("pushed_at", "")
            if item.get("stargazers_count", 0) > row["stars"]:
                row["stars"] = item.get("stargazers_count", 0)

    rows = []
    for row in repos.values():
        rows.append(
            {
                "repo": row["repo"],
                "stars": row["stars"],
                "updated_at": row["updated_at"],
                "pushed_at": row["pushed_at"],
                "signals": ",".join(sorted(row["signals"])),
            }
        )
    rows.sort(key=lambda x: (x["updated_at"], x["stars"]), reverse=True)
    return rows[:15]


def fetch_openclaw_highlights():
    readme = gh_api("repos/openclaw/openclaw/readme")
    raw = base64.b64decode(readme["content"]).decode("utf-8", errors="ignore")
    lines = raw.splitlines()
    out = []
    in_section = False
    for line in lines:
        if line.strip() == "## Highlights":
            in_section = True
            continue
        if in_section and line.startswith("## "):
            break
        if in_section and line.strip().startswith("- **"):
            out.append(line.strip())
        if len(out) >= 10:
            break
    return out


def fetch_openclaw_recent_prs():
    pulls = gh_api("repos/openclaw/openclaw/pulls?state=open&sort=updated&direction=desc&per_page=20")
    rows = []
    for pr in pulls[:12]:
        rows.append(
            {
                "number": pr.get("number"),
                "title": pr.get("title", ""),
                "url": pr.get("html_url", ""),
                "labels": [x.get("name", "") for x in pr.get("labels", [])],
                "updated_at": pr.get("updated_at", ""),
            }
        )
    return rows


def parse_alter0_requirements():
    text = REQ_FILE.read_text(encoding="utf-8")
    rows = []
    for line in text.splitlines():
        line = line.strip()
        if not line.startswith("| R-"):
            continue
        cols = [c.strip() for c in line.strip("|").split("|")]
        if len(cols) < 4:
            continue
        rows.append({"id": cols[0], "name": cols[1], "status": cols[2], "desc": cols[3]})
    return rows


def short_iso(ts):
    if not ts:
        return "-"
    return ts.replace("T", " ").replace("Z", " UTC")


def build_report(today, stats, hotspots, highlights, prs, requirements):
    now_cst = dt.datetime.now(SH_TZ).strftime("%Y-%m-%d %H:%M:%S %Z")
    supported = [r for r in requirements if r["status"] == "supported"]
    planned = [r for r in requirements if r["status"] == "planned"]

    label_counter = Counter()
    for pr in prs:
        for label in pr["labels"]:
            label_counter[label] += 1

    top_labels = ", ".join([f"{k}({v})" for k, v in label_counter.most_common(8)]) or "暂无标签数据"

    lines = []
    lines.append(f"# AI Frontier Research - {today}")
    lines.append("")
    lines.append(f"> Generated at: {now_cst}")
    lines.append("")
    lines.append("## 1) 行业前沿快照（核心样本）")
    lines.append("")
    lines.append("| Repo | Stars | Last Push | Last Update |")
    lines.append("| --- | ---: | --- | --- |")
    for row in sorted(stats, key=lambda x: x["stars"], reverse=True):
        lines.append(
            f"| `{row['repo']}` | {row['stars']} | {short_iso(row['pushed_at'])} | {short_iso(row['updated_at'])} |"
        )
    lines.append("")

    lines.append("## 2) 自动抓取 AI 热点（不限 AI Agent/AI Coding/OpenClaw）")
    lines.append("")
    lines.append("| Repo | Signals | Stars | Last Push | Last Update |")
    lines.append("| --- | --- | ---: | --- | --- |")
    for row in hotspots[:12]:
        lines.append(
            f"| `{row['repo']}` | `{row['signals']}` | {row['stars']} | {short_iso(row['pushed_at'])} | {short_iso(row['updated_at'])} |"
        )
    lines.append("")
    lines.append("趋势观察：")
    lines.append("1. 热点采样来源于多查询自动抓取（agent/coding/mcp/rag/infra/multimodal），不再绑定固定少量主题。")
    lines.append("2. 高活跃仓库以最近 14 天持续更新为主，能够反映当前工程落地节奏。")
    lines.append("3. 结合 OpenClaw 社区 PR 热点，可形成“外部趋势 + 内部需求”双向对照。")
    lines.append("")

    lines.append("## 3) OpenClaw 已支持能力（抽样）")
    lines.append("")
    for item in highlights[:8]:
        lines.append(f"- {item[2:] if item.startswith('- ') else item}")
    lines.append("")

    lines.append("## 4) OpenClaw 社区近期需求信号（来自开放 PR）")
    lines.append("")
    lines.append(f"- 热点标签：{top_labels}")
    for pr in prs[:10]:
        lines.append(f"- #{pr['number']} {pr['title']} ({pr['url']})")
    lines.append("")

    lines.append("## 5) 与 alter0 能力对比")
    lines.append("")
    lines.append(f"- alter0 当前已支持需求：{len(supported)} 项；规划中：{len(planned)} 项。")
    lines.append("- 对比结论：")
    lines.append("  - 通道与生态：OpenClaw 为多通道成熟能力；alter0 当前聚焦 CLI/Web + 基础 Control。")
    lines.append("  - 编排与工具：OpenClaw 已形成工具平台（browser/canvas/nodes/cron）；alter0 在 Skills/MCP 标准化与接入阶段（R-021~R-023）。")
    lines.append("  - 记忆体系：OpenClaw 具备生产级会话运行体系；alter0 已明确分层记忆路线（R-017~R-026），待实现。")
    lines.append("  - Web 体验：alter0 已明确移动端、流式、侧栏交互优化路线（R-012~R-015），具备快速补齐基础。")
    lines.append("")

    lines.append("## 6) 对 alter0 的建议落地顺序")
    lines.append("")
    lines.append("1. 先完成 R-013 + R-016（流式 + 并发治理），保障核心交互稳定性。")
    lines.append("2. 并行推进 R-021~R-023（Skills/MCP 标准化与 Codex 接入），形成扩展能力基座。")
    lines.append("3. 按 R-017~R-026 构建记忆栈，优先做天级归档与 L1/L2/L3 分层策略。")
    lines.append("4. 持续跟踪 OpenClaw 社区 PR 热点，每日同步差距清单到本目录研究报告。")
    lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def update_research_index(today):
    index_path = RESEARCH_DIR / "README.md"
    old = index_path.read_text(encoding="utf-8") if index_path.exists() else "# Research Reports\n\n"
    links = []
    for line in old.splitlines():
        m = re.match(r"^- \[(\d{4}-\d{2}-\d{2})\]\(\./\1\.md\)$", line.strip())
        if m:
            links.append(m.group(1))
    if today not in links:
        links.append(today)
    links = sorted(set(links), reverse=True)[:90]

    out = ["# Research Reports", "", "自动生成的 AI 前沿研究报告索引。", ""]
    for d in links:
        out.append(f"- [{d}](./{d}.md)")
    out.append("")
    index_path.write_text("\n".join(out), encoding="utf-8")


def ensure_docs_index_link():
    text = DOCS_INDEX.read_text(encoding="utf-8")
    line = "5. [研究报告](./research/README.md)"
    if line in text:
        return
    text = text.replace(
        "4. [GitHub App 提交流程（Verified）](./github-app-submission.md)",
        "4. [GitHub App 提交流程（Verified）](./github-app-submission.md)\n5. [研究报告](./research/README.md)",
    )
    DOCS_INDEX.write_text(text, encoding="utf-8")


def get_remote_file(owner_repo, branch, path):
    safe_path = path.replace(" ", "%20")
    try:
        data = gh_api(f"repos/{owner_repo}/contents/{safe_path}?ref={branch}")
    except Exception:
        return None
    if not data or "content" not in data:
        return None
    content = base64.b64decode(data["content"]).decode("utf-8", errors="ignore")
    return {"sha": data.get("sha"), "content": content}


def upsert_file(owner_repo, branch, local_path, commit_message):
    rel = local_path.relative_to(ROOT).as_posix()
    local_text = local_path.read_text(encoding="utf-8")
    remote = get_remote_file(owner_repo, branch, rel)
    if remote and remote["content"] == local_text:
        return False

    fields = {
        "message": commit_message,
        "content": base64.b64encode(local_text.encode("utf-8")).decode("ascii"),
        "branch": branch,
    }
    if remote and remote.get("sha"):
        fields["sha"] = remote["sha"]
    gh_api(f"repos/{owner_repo}/contents/{rel}", method="PUT", fields=fields)
    return True


def ensure_branch(owner_repo, base_branch, branch):
    base_sha = gh_api(f"repos/{owner_repo}/git/ref/heads/{base_branch}")["object"]["sha"]
    ref_path = f"repos/{owner_repo}/git/ref/heads/{branch}"
    try:
        gh_api(ref_path)
        gh_api(f"repos/{owner_repo}/git/refs/heads/{branch}", method="PATCH", fields={"sha": base_sha, "force": "true"})
    except Exception:
        gh_api(f"repos/{owner_repo}/git/refs", method="POST", fields={"ref": f"refs/heads/{branch}", "sha": base_sha})


def ensure_pr(owner_repo, base_branch, branch, title, body, assignee):
    existing = json.loads(run(["gh", "pr", "list", "--repo", owner_repo, "--head", branch, "--state", "open", "--json", "number,url"]))
    if existing:
        pr_number = existing[0]["number"]
        pr_url = existing[0]["url"]
    else:
        pr_url = run([
            "gh", "pr", "create",
            "--repo", owner_repo,
            "--base", base_branch,
            "--head", branch,
            "--title", title,
            "--body", body,
        ])
        pr_number = int(pr_url.rstrip("/").split("/")[-1])
    if assignee:
        run(["gh", "pr", "edit", str(pr_number), "--repo", owner_repo, "--add-assignee", assignee], check=False)
        run(["gh", "pr", "edit", str(pr_number), "--repo", owner_repo, "--add-reviewer", assignee], check=False)
    return pr_number, pr_url


def merge_pr(owner_repo, pr_number):
    cmd = ["gh", "pr", "merge", str(pr_number), "--repo", owner_repo, "--squash", "--delete-branch"]
    p = subprocess.run(cmd, cwd=ROOT, text=True, capture_output=True)
    if p.returncode == 0:
        return True, "merged"
    cmd_auto = ["gh", "pr", "merge", str(pr_number), "--repo", owner_repo, "--auto", "--squash", "--delete-branch"]
    p2 = subprocess.run(cmd_auto, cwd=ROOT, text=True, capture_output=True)
    if p2.returncode == 0:
        return True, "auto-merge-enabled"
    return False, (p.stderr + "\n" + p2.stderr).strip()


def main():
    owner_repo = parse_owner_repo()
    base_branch = os.getenv("RESEARCH_BASE_BRANCH", "master")
    assignee = os.getenv("RESEARCH_ASSIGNEE", "zhangqian-silk")

    now = dt.datetime.now(SH_TZ)
    today = now.strftime("%Y-%m-%d")
    ymd = now.strftime("%Y%m%d")
    branch = f"research/daily-{ymd}"

    RESEARCH_DIR.mkdir(parents=True, exist_ok=True)
    stats = fetch_repo_stats()
    hotspots = fetch_ai_hotspots()
    highlights = fetch_openclaw_highlights()
    prs = fetch_openclaw_recent_prs()
    requirements = parse_alter0_requirements()

    report = build_report(today, stats, hotspots, highlights, prs, requirements)
    report_path = RESEARCH_DIR / f"{today}.md"
    report_path.write_text(report, encoding="utf-8")
    update_research_index(today)
    ensure_docs_index_link()

    tracked_files = [
        report_path,
        RESEARCH_DIR / "README.md",
        DOCS_INDEX,
        ROOT / "scripts/research/daily_research_report.py",
        ROOT / "scripts/research/run_daily_research.sh",
    ]

    ensure_branch(owner_repo, base_branch, branch)

    changed = 0
    for file_path in tracked_files:
        msg = f"docs(research): update daily report {today} ({file_path.relative_to(ROOT).as_posix()})"
        if upsert_file(owner_repo, branch, file_path, msg):
            changed += 1

    if changed == 0:
        print("No changes detected; skip PR")
        return

    title = f"docs(research): daily AI frontier report {today}"
    body = (
        "## What\n"
        "- refresh daily AI frontier report under `docs/research/`\n"
        "- sync OpenClaw capability/trend snapshot and compare against alter0 roadmap\n"
        "- keep research automation pipeline docs/scripts up to date\n\n"
        "## Notes\n"
        "- generated by scheduled research task\n"
        "- this PR is intended to auto-merge when checks pass\n"
    )

    pr_number, pr_url = ensure_pr(owner_repo, base_branch, branch, title, body, assignee)
    ok, merge_msg = merge_pr(owner_repo, pr_number)
    print(json.dumps({"pr": pr_url, "changed_files": changed, "merged": ok, "detail": merge_msg}, ensure_ascii=False))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)
