#!/usr/bin/env python3
import base64
import datetime as dt
import json
import os
import re
import subprocess
import sys
import time
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from collections import Counter
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DOCS_DIR = ROOT / "docs"
RESEARCH_DIR = DOCS_DIR / "research"
HOTSPOT_DIR = RESEARCH_DIR / "hotspots"
COMPARE_DIR = RESEARCH_DIR / "comparison"
REQ_FILE = DOCS_DIR / "requirements.md"
DOCS_INDEX = DOCS_DIR / "README.md"

SH_TZ = dt.timezone(dt.timedelta(hours=8))

# 额外保留少量 GitHub 仓库热度作为补充，不作为唯一来源。
REPO_CANDIDATES = [
    "openai/codex",
    "anthropics/claude-code",
    "Aider-AI/aider",
    "langchain-ai/langgraph",
    "openai/openai-agents-python",
    "modelcontextprotocol/modelcontextprotocol",
    "openclaw/openclaw",
]

BLOG_RSS_SOURCES = [
    ("OpenAI News", "blog", "https://openai.com/news/rss.xml"),
    ("Google AI Blog", "blog", "https://blog.google/technology/ai/rss/"),
    ("Hugging Face Blog", "blog", "https://huggingface.co/blog/feed.xml"),
    ("The Verge AI", "website", "https://www.theverge.com/rss/ai-artificial-intelligence/index.xml"),
    ("MIT Tech Review AI", "website", "https://www.technologyreview.com/topic/artificial-intelligence/feed"),
]

TOPIC_KEYWORDS = {
    "Agent编排": ["agent", "multi-agent", "workflow", "orchestration"],
    "AI Coding": ["coding", "code", "copilot", "codex", "ide"],
    "推理与模型": ["reasoning", "inference", "model", "llm", "benchmark"],
    "MCP/工具生态": ["mcp", "tool", "protocol", "sdk", "connector"],
    "RAG与记忆": ["rag", "retrieval", "memory", "vector", "knowledge"],
    "多模态": ["vision", "audio", "video", "multimodal", "speech"],
}


def run(cmd, cwd=None, check=True):
    p = subprocess.run(cmd, cwd=cwd or ROOT, text=True, capture_output=True)
    if check and p.returncode != 0:
        raise RuntimeError(f"cmd failed: {' '.join(cmd)}\n{p.stderr.strip()}")
    return p.stdout.strip()


def gh_api(path, method="GET", fields=None, check=True):
    cmd = ["gh", "api", path]
    if method != "GET":
        cmd += ["-X", method]
    if fields:
        for key, value in fields.items():
            cmd += ["-f", f"{key}={value}"]
    p = subprocess.run(cmd, cwd=ROOT, text=True, capture_output=True)
    if check and p.returncode != 0:
        raise RuntimeError(f"gh api failed: {path}\n{p.stderr.strip()}")
    if p.returncode != 0:
        return None
    out = p.stdout.strip()
    return json.loads(out) if out else None


def parse_owner_repo():
    env_repo = os.getenv("ALTER0_REPO")
    if env_repo and "/" in env_repo:
        return env_repo
    url = run(["git", "remote", "get-url", "origin"])
    m = re.search(r"[:/]([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+?)(?:\.git)?$", url)
    if not m:
        return "zhangqian-silk/alter0"
    return m.group(1)


def fetch_url(url, timeout=20):
    req = urllib.request.Request(
        url,
        headers={
            "User-Agent": "alter0-research-bot/1.0 (+https://github.com/zhangqian-silk/alter0)",
            "Accept": "application/json,application/rss+xml,application/xml,text/xml,text/html,*/*",
        },
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        data = resp.read()
        return data.decode("utf-8", errors="ignore")


def parse_time_to_str(raw):
    if not raw:
        return "-"
    raw = raw.strip()
    # 尽量保留源格式，同时将 RFC822 / ISO 时间规整到统一展示。
    try:
        for fmt in (
            "%a, %d %b %Y %H:%M:%S %z",
            "%a, %d %b %Y %H:%M:%S %Z",
            "%Y-%m-%dT%H:%M:%S%z",
            "%Y-%m-%dT%H:%M:%SZ",
            "%Y-%m-%d %H:%M:%S",
        ):
            try:
                dt_obj = dt.datetime.strptime(raw, fmt)
                return dt_obj.astimezone(SH_TZ).strftime("%Y-%m-%d %H:%M")
            except ValueError:
                continue
    except Exception:
        pass
    return raw[:19]


def fetch_rss_items(name, kind, url, limit=8):
    items = []
    try:
        xml_text = fetch_url(url)
        root = ET.fromstring(xml_text)

        # RSS 2.0
        for it in root.findall(".//item")[:limit]:
            title = (it.findtext("title") or "").strip()
            link = (it.findtext("link") or "").strip()
            pub = (it.findtext("pubDate") or it.findtext("published") or "").strip()
            if not title or not link:
                continue
            items.append(
                {
                    "source": name,
                    "kind": kind,
                    "title": title,
                    "url": link,
                    "published": parse_time_to_str(pub),
                }
            )

        # Atom
        ns = {"a": "http://www.w3.org/2005/Atom"}
        for it in root.findall(".//a:entry", ns)[:limit]:
            title = (it.findtext("a:title", default="", namespaces=ns) or "").strip()
            link_el = it.find("a:link", ns)
            link = (link_el.attrib.get("href") if link_el is not None else "") or ""
            pub = (it.findtext("a:updated", default="", namespaces=ns) or "").strip()
            if not title or not link:
                continue
            items.append(
                {
                    "source": name,
                    "kind": kind,
                    "title": title,
                    "url": link,
                    "published": parse_time_to_str(pub),
                }
            )
    except Exception:
        return []

    return items[:limit]


def fetch_hn_items(limit=15):
    queries = ["AI", "LLM", "agent", "MCP", "RAG", "coding assistant"]
    out = []
    seen = set()
    cutoff = int(time.time()) - 7 * 24 * 3600

    for q in queries:
        params = urllib.parse.urlencode(
            {
                "query": q,
                "tags": "story",
                "hitsPerPage": 8,
                "numericFilters": f"created_at_i>{cutoff}",
            }
        )
        url = f"https://hn.algolia.com/api/v1/search_by_date?{params}"
        try:
            payload = json.loads(fetch_url(url))
        except Exception:
            continue

        for hit in payload.get("hits", []):
            title = (hit.get("title") or "").strip()
            link = (hit.get("url") or "").strip()
            if not title or not link:
                continue
            key = (title, link)
            if key in seen:
                continue
            seen.add(key)
            out.append(
                {
                    "source": "Hacker News",
                    "kind": "forum",
                    "title": title,
                    "url": link,
                    "published": parse_time_to_str(hit.get("created_at", "")),
                }
            )
            if len(out) >= limit:
                return out

    return out


def fetch_reddit_items(limit=10):
    url = "https://www.reddit.com/r/MachineLearning/hot.json?limit=25"
    items = []
    try:
        payload = json.loads(fetch_url(url))
        for child in payload.get("data", {}).get("children", []):
            data = child.get("data", {})
            title = (data.get("title") or "").strip()
            permalink = data.get("permalink") or ""
            link = f"https://www.reddit.com{permalink}" if permalink else ""
            created = data.get("created_utc")
            if not title or not link:
                continue
            if any(k in title.lower() for k in ["hiring", "weekly", "discussion"]):
                continue
            published = "-"
            if created:
                published = dt.datetime.fromtimestamp(created, tz=SH_TZ).strftime("%Y-%m-%d %H:%M")
            items.append(
                {
                    "source": "Reddit r/MachineLearning",
                    "kind": "forum",
                    "title": title,
                    "url": link,
                    "published": published,
                }
            )
            if len(items) >= limit:
                break
    except Exception:
        return []
    return items


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
        if len(out) >= 8:
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


def topic_of(title):
    low = title.lower()
    hits = []
    for topic, kws in TOPIC_KEYWORDS.items():
        if any(k in low for k in kws):
            hits.append(topic)
    return hits or ["其他"]


def collect_hotspots():
    items = []

    # 论坛
    items.extend(fetch_hn_items())
    items.extend(fetch_reddit_items())

    # 博客/网站
    for name, kind, url in BLOG_RSS_SOURCES:
        items.extend(fetch_rss_items(name, kind, url, limit=8))

    # 去重
    dedup = {}
    for it in items:
        key = (it["title"].strip(), it["url"].strip())
        if key not in dedup:
            dedup[key] = it
    out = list(dedup.values())

    # 按发布时间字符串近似排序（已规整为 yyyy-mm-dd hh:mm 的优先）
    out.sort(key=lambda x: x.get("published", ""), reverse=True)
    return out[:60]


def build_hotspot_report(today, hotspots, repo_stats):
    now_cst = dt.datetime.now(SH_TZ).strftime("%Y-%m-%d %H:%M:%S")

    kind_counter = Counter([x["kind"] for x in hotspots])
    topic_counter = Counter()
    source_counter = Counter()
    for h in hotspots:
        source_counter[h["source"]] += 1
        for t in topic_of(h["title"]):
            topic_counter[t] += 1

    top_topics = "、".join([f"{k}({v})" for k, v in topic_counter.most_common(6)]) or "暂无"
    top_sources = "、".join([f"{k}({v})" for k, v in source_counter.most_common(8)]) or "暂无"

    lines = []
    lines.append(f"# AI 热点追踪（多来源）- {today}")
    lines.append("")
    lines.append(f"> 生成时间：{now_cst} (Asia/Shanghai)")
    lines.append("")
    lines.append("## 1) 抓取范围与统计")
    lines.append("")
    lines.append(f"- 本次抓取总条目：{len(hotspots)}")
    lines.append(f"- 来源结构：论坛 {kind_counter.get('forum',0)} / 博客 {kind_counter.get('blog',0)} / 网站 {kind_counter.get('website',0)}")
    lines.append(f"- 热点主题：{top_topics}")
    lines.append(f"- 活跃来源：{top_sources}")
    lines.append("")
    lines.append("## 2) 中文趋势总结")
    lines.append("")
    lines.append("1. 热点来源已扩展到论坛、博客和技术网站，不再只依赖 Git 仓库活跃度。")
    lines.append("2. 近期讨论集中在 Agent 编排、AI Coding 工具链、推理优化与多模态集成。")
    lines.append("3. MCP/工具互联与 RAG/记忆话题持续高频，说明生态从“模型能力”转向“系统能力”。")
    lines.append("")
    lines.append("## 3) 热点明细（多来源）")
    lines.append("")
    lines.append("| 来源 | 类型 | 发布时间 | 标题 | 链接 |")
    lines.append("| --- | --- | --- | --- | --- |")
    for h in hotspots[:40]:
        title = h["title"].replace("|", "-")
        lines.append(f"| {h['source']} | {h['kind']} | {h['published']} | {title} | [link]({h['url']}) |")
    lines.append("")
    lines.append("## 4) GitHub 热度样本（补充）")
    lines.append("")
    lines.append("| Repo | Stars | 最近推送 | 最近更新 |")
    lines.append("| --- | ---: | --- | --- |")
    for row in sorted(repo_stats, key=lambda x: x["stars"], reverse=True):
        lines.append(
            f"| `{row['repo']}` | {row['stars']} | {row['pushed_at'].replace('T',' ')[:16]} | {row['updated_at'].replace('T',' ')[:16]} |"
        )
    lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def build_compare_report(today, hotspots, requirements, highlights, prs):
    now_cst = dt.datetime.now(SH_TZ).strftime("%Y-%m-%d %H:%M:%S")

    status_counter = Counter([r["status"] for r in requirements])

    topic_counter = Counter()
    for h in hotspots:
        for t in topic_of(h["title"]):
            topic_counter[t] += 1

    label_counter = Counter()
    for pr in prs:
        for lb in pr["labels"]:
            label_counter[lb] += 1

    # 粗粒度映射：外部主题 -> alter0 相关需求
    topic_to_req = {
        "Agent编排": ["R-016", "R-021", "R-022"],
        "AI Coding": ["R-022", "R-023"],
        "MCP/工具生态": ["R-021", "R-023"],
        "RAG与记忆": ["R-017", "R-018", "R-019", "R-020", "R-024", "R-025", "R-026"],
        "多模态": ["R-014", "R-015"],
        "推理与模型": ["R-013", "R-016"],
    }
    req_map = {r["id"]: r for r in requirements}

    lines = []
    lines.append(f"# OpenClaw / alter0 功能对比 - {today}")
    lines.append("")
    lines.append(f"> 生成时间：{now_cst} (Asia/Shanghai)")
    lines.append("")
    lines.append("## 1) OpenClaw 当前能力（抽样）")
    lines.append("")
    for item in highlights[:8]:
        lines.append(f"- {item[2:] if item.startswith('- ') else item}")
    lines.append("")

    lines.append("## 2) OpenClaw 社区需求信号（开放 PR）")
    lines.append("")
    lines.append("- 热点标签：" + ("、".join([f"{k}({v})" for k, v in label_counter.most_common(8)]) or "暂无"))
    for pr in prs[:10]:
        lines.append(f"- #{pr['number']} {pr['title']} ({pr['url']})")
    lines.append("")

    lines.append("## 3) alter0 需求覆盖现状")
    lines.append("")
    lines.append(
        f"- 需求状态分布：supported={status_counter.get('supported',0)} / ready={status_counter.get('ready',0)} / planned={status_counter.get('planned',0)}"
    )
    lines.append("| 外部热点主题 | 热度 | alter0 对应需求 | 覆盖状态 |")
    lines.append("| --- | ---: | --- | --- |")

    for topic, cnt in topic_counter.most_common(8):
        req_ids = topic_to_req.get(topic, [])
        if req_ids:
            refs = []
            stats = Counter()
            for rid in req_ids:
                row = req_map.get(rid)
                if row:
                    refs.append(f"{rid}({row['status']})")
                    stats[row["status"]] += 1
            coverage = f"supported:{stats.get('supported',0)} ready:{stats.get('ready',0)} planned:{stats.get('planned',0)}"
            lines.append(f"| {topic} | {cnt} | {', '.join(refs)} | {coverage} |")
        else:
            lines.append(f"| {topic} | {cnt} | - | 暂无直接映射 |")
    lines.append("")

    lines.append("## 4) 差距判断与优先建议")
    lines.append("")
    lines.append("1. 热点持续聚焦系统化能力（编排/并发/记忆/MCP），alter0 已有路线但实现覆盖仍偏前期。")
    lines.append("2. 优先推进能够同时提升稳定性与体验的组合：R-013 + R-016 + R-015。")
    lines.append("3. 并行建设生态能力底座：R-021~R-023（Skills/MCP 标准化与接入）。")
    lines.append("4. 记忆体系建议按短期到长期分层落地：R-017 -> R-019 -> R-024/R-025/R-026。")
    lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def update_research_index(today):
    RESEARCH_DIR.mkdir(parents=True, exist_ok=True)
    HOTSPOT_DIR.mkdir(parents=True, exist_ok=True)
    COMPARE_DIR.mkdir(parents=True, exist_ok=True)

    # 扫目录自动索引
    hotspot_dates = sorted([p.stem for p in HOTSPOT_DIR.glob("*.md") if re.match(r"\d{4}-\d{2}-\d{2}", p.stem)], reverse=True)[:120]
    compare_dates = sorted([p.stem for p in COMPARE_DIR.glob("*.md") if re.match(r"\d{4}-\d{2}-\d{2}", p.stem)], reverse=True)[:120]

    if today not in hotspot_dates:
        hotspot_dates.insert(0, today)
    if today not in compare_dates:
        compare_dates.insert(0, today)

    lines = [
        "# Research Reports",
        "",
        "自动生成的 AI 前沿研究文档索引（热点追踪与能力对比分离）。",
        "",
        "## 热点追踪文档",
        "",
    ]
    for d in hotspot_dates[:90]:
        lines.append(f"- [{d}](./hotspots/{d}.md)")

    lines.extend(["", "## 功能对比文档", ""])
    for d in compare_dates[:90]:
        lines.append(f"- [{d}](./comparison/{d}.md)")

    lines.append("")
    (RESEARCH_DIR / "README.md").write_text("\n".join(lines), encoding="utf-8")


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
    data = gh_api(f"repos/{owner_repo}/contents/{safe_path}?ref={branch}", check=False)
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
    if gh_api(ref_path, check=False):
        gh_api(f"repos/{owner_repo}/git/refs/heads/{branch}", method="PATCH", fields={"sha": base_sha, "force": "true"})
    else:
        gh_api(f"repos/{owner_repo}/git/refs", method="POST", fields={"ref": f"refs/heads/{branch}", "sha": base_sha})


def ensure_pr(owner_repo, base_branch, branch, title, body, assignee):
    existing = json.loads(
        run(["gh", "pr", "list", "--repo", owner_repo, "--head", branch, "--state", "open", "--json", "number,url"])
    )
    if existing:
        pr_number = existing[0]["number"]
        pr_url = existing[0]["url"]
    else:
        pr_url = run(
            [
                "gh",
                "pr",
                "create",
                "--repo",
                owner_repo,
                "--base",
                base_branch,
                "--head",
                branch,
                "--title",
                title,
                "--body",
                body,
            ]
        )
        pr_number = int(pr_url.rstrip("/").split("/")[-1])

    add_assignee = os.getenv("AUTO_MERGE_ADD_ASSIGNEE", "0") == "1"
    add_reviewer = os.getenv("AUTO_MERGE_ADD_REVIEWER", "0") == "1"
    if assignee and add_assignee:
        run(["gh", "pr", "edit", str(pr_number), "--repo", owner_repo, "--add-assignee", assignee], check=False)
    if assignee and add_reviewer:
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

    HOTSPOT_DIR.mkdir(parents=True, exist_ok=True)
    COMPARE_DIR.mkdir(parents=True, exist_ok=True)

    hotspots = collect_hotspots()
    repo_stats = fetch_repo_stats()
    requirements = parse_alter0_requirements()
    highlights = fetch_openclaw_highlights()
    prs = fetch_openclaw_recent_prs()

    hotspot_doc = build_hotspot_report(today, hotspots, repo_stats)
    compare_doc = build_compare_report(today, hotspots, requirements, highlights, prs)

    hotspot_path = HOTSPOT_DIR / f"{today}.md"
    compare_path = COMPARE_DIR / f"{today}.md"
    hotspot_path.write_text(hotspot_doc, encoding="utf-8")
    compare_path.write_text(compare_doc, encoding="utf-8")

    update_research_index(today)
    ensure_docs_index_link()

    tracked_files = [
        hotspot_path,
        compare_path,
        RESEARCH_DIR / "README.md",
        DOCS_INDEX,
        ROOT / "scripts/research/daily_research_report.py",
        ROOT / "scripts/research/run_daily_research.sh",
    ]

    ensure_branch(owner_repo, base_branch, branch)

    changed = 0
    for file_path in tracked_files:
        msg = f"docs(research): update research docs {today} ({file_path.relative_to(ROOT).as_posix()})"
        if upsert_file(owner_repo, branch, file_path, msg):
            changed += 1

    if changed == 0:
        print("No changes detected; skip PR")
        return

    title = f"docs(research): daily hotspots + comparison {today}"
    body = (
        "## What\n"
        "- refresh multi-source AI hotspots report (forum/blog/website + repo samples)\n"
        "- refresh OpenClaw vs alter0 comparison report in a separate document\n"
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
