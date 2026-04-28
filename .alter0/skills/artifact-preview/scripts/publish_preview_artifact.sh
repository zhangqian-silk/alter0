#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: .alter0/skills/artifact-preview/scripts/publish_preview_artifact.sh <session_id> [service_name] [options]

Build and deploy a static preview page for text, image, and code artifacts.

Options:
  --artifact <path>       File to include in the preview. Repeat for multiple files.
  --title <text>          Preview page title. Default: Artifact Preview
  --description <text>    Optional summary rendered at the top of the page.
  --repo-path <path>      Repository root. Defaults to the current alter0 repo.
  --health-path <path>    Health probe path for the preview service. Default: /
  -h, --help              Show this help.
EOF
}

SESSION_ID="${1:-}"
if [[ -z "${SESSION_ID}" || "${SESSION_ID}" == "-h" || "${SESSION_ID}" == "--help" ]]; then
  usage
  exit 0
fi
shift

SERVICE_NAME="preview"
if [[ $# -gt 0 && "${1#-}" == "${1}" ]]; then
  SERVICE_NAME="$1"
  shift
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_PATH="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
TITLE="Artifact Preview"
DESCRIPTION=""
HEALTH_PATH="/"
ARTIFACTS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --artifact)
      ARTIFACTS+=("${2:-}")
      shift 2
      ;;
    --title)
      TITLE="${2:-}"
      shift 2
      ;;
    --description)
      DESCRIPTION="${2:-}"
      shift 2
      ;;
    --repo-path)
      REPO_PATH="${2:-}"
      shift 2
      ;;
    --health-path)
      HEALTH_PATH="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ "${#ARTIFACTS[@]}" -eq 0 ]]; then
  echo "at least one --artifact path is required" >&2
  exit 1
fi

if [[ ! "${SERVICE_NAME}" =~ ^[a-z0-9-]+$ ]]; then
  echo "service_name must match ^[a-z0-9-]+$" >&2
  exit 1
fi
if [[ "${SERVICE_NAME}" == "web" || "${SERVICE_NAME}" == "travel" ]]; then
  echo "service_name ${SERVICE_NAME} is reserved; use a dedicated preview subdomain" >&2
  exit 1
fi

REPO_PATH="$(cd "${REPO_PATH}" && pwd)"
SITE_ROOT="${REPO_PATH}/.alter0/preview-artifacts/${SERVICE_NAME}"
SITE_DIR="${SITE_ROOT}/site"

python3 - "${SITE_DIR}" "${TITLE}" "${DESCRIPTION}" "${ARTIFACTS[@]}" <<'PY'
import html
import mimetypes
import os
import shutil
import sys
from pathlib import Path
from urllib.parse import quote

site_dir = Path(sys.argv[1])
title = sys.argv[2].strip() or "Artifact Preview"
description = sys.argv[3].strip()
artifact_args = [Path(item).expanduser().resolve() for item in sys.argv[4:]]

for artifact in artifact_args:
    if not artifact.exists():
        raise SystemExit(f"artifact not found: {artifact}")
    if artifact.is_dir():
        raise SystemExit(f"directories are not supported: {artifact}")

if site_dir.exists():
    shutil.rmtree(site_dir)
(site_dir / "files").mkdir(parents=True, exist_ok=True)

code_suffixes = {
    ".c", ".cc", ".cpp", ".cs", ".css", ".go", ".graphql", ".h", ".hpp",
    ".html", ".java", ".js", ".json", ".jsx", ".kt", ".less", ".lua",
    ".mjs", ".md", ".php", ".py", ".rb", ".rs", ".sass", ".scss", ".sh",
    ".sql", ".svg", ".swift", ".toml", ".tsx", ".ts", ".txt", ".xml",
    ".yaml", ".yml",
}

used_names = {}
sections = []
nav_items = []


def unique_name(name: str) -> str:
    stem, suffix = os.path.splitext(name)
    count = used_names.get(name, 0)
    if count == 0:
        used_names[name] = 1
        return name
    while True:
        candidate = f"{stem}-{count + 1}{suffix}"
        if candidate not in used_names:
            used_names[name] = count + 1
            used_names[candidate] = 1
            return candidate
        count += 1


for idx, artifact in enumerate(artifact_args, start=1):
    safe_name = unique_name(artifact.name)
    target = site_dir / "files" / safe_name
    shutil.copy2(artifact, target)
    mime, _ = mimetypes.guess_type(target.name)
    mime = mime or "application/octet-stream"
    suffix = artifact.suffix.lower()
    href = "files/" + quote(safe_name)
    section_id = f"artifact-{idx}"

    preview = [f'<p class="artifact-links"><a href="{href}">Open raw file</a></p>']
    if mime.startswith("image/"):
        preview.append(
            f'<figure class="artifact-image"><img src="{href}" alt="{html.escape(artifact.name)}"></figure>'
        )
    elif mime.startswith("text/") or suffix in code_suffixes:
        raw = artifact.read_text(encoding="utf-8", errors="replace")
        truncated = False
        if len(raw) > 200000:
            raw = raw[:200000]
            truncated = True
        preview.append('<pre class="artifact-code"><code>')
        preview.append(html.escape(raw))
        preview.append("</code></pre>")
        if truncated:
            preview.append('<p class="artifact-note">Preview truncated after 200000 characters.</p>')
    else:
        preview.append('<p class="artifact-note">Binary preview is not rendered inline. Use the raw file link.</p>')

    nav_items.append(
        f'<li><a href="#{section_id}">{html.escape(artifact.name)}</a><span>{html.escape(mime)}</span></li>'
    )
    sections.append(
        "\n".join(
            [
                f'<section id="{section_id}" class="artifact-section">',
                f'  <header><h2>{html.escape(artifact.name)}</h2><p>{html.escape(mime)}</p></header>',
                *[f"  {line}" for line in preview],
                "</section>",
            ]
        )
    )

description_html = f"<p class=\"page-description\">{html.escape(description)}</p>" if description else ""

document = f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{html.escape(title)}</title>
  <style>
    :root {{
      color-scheme: light;
      --bg: #f4efe7;
      --surface: rgba(255, 252, 248, 0.92);
      --surface-strong: #fffaf2;
      --text: #1f1a17;
      --muted: #75685f;
      --accent: #b04d1e;
      --border: rgba(78, 54, 39, 0.16);
      --shadow: 0 24px 60px rgba(80, 58, 43, 0.14);
      --font-display: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", serif;
      --font-body: "IBM Plex Sans", "Segoe UI", sans-serif;
      --font-code: "IBM Plex Mono", "SFMono-Regular", monospace;
    }}
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0;
      min-height: 100vh;
      background:
        radial-gradient(circle at top left, rgba(176, 77, 30, 0.12), transparent 34%),
        linear-gradient(180deg, #f7f1e8 0%, #efe6da 100%);
      color: var(--text);
      font-family: var(--font-body);
      line-height: 1.6;
    }}
    main {{
      width: min(1080px, calc(100vw - 32px));
      margin: 0 auto;
      padding: 40px 0 72px;
    }}
    .hero {{
      padding: 28px 28px 18px;
      border-bottom: 1px solid var(--border);
    }}
    .hero-shell,
    .artifact-section,
    .artifact-nav {{
      background: var(--surface);
      border: 1px solid var(--border);
      box-shadow: var(--shadow);
      backdrop-filter: blur(8px);
    }}
    .hero-shell {{
      overflow: hidden;
      border-radius: 28px 28px 18px 18px;
      margin-bottom: 18px;
    }}
    h1, h2 {{
      margin: 0;
      font-family: var(--font-display);
      font-weight: 600;
      letter-spacing: -0.03em;
    }}
    h1 {{
      font-size: clamp(2.2rem, 5vw, 4.2rem);
      line-height: 0.95;
      max-width: 12ch;
    }}
    .eyebrow {{
      margin: 0 0 12px;
      font-size: 0.8rem;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      color: var(--accent);
    }}
    .hero-meta {{
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
      margin-top: 18px;
      color: var(--muted);
      font-size: 0.95rem;
    }}
    .page-description {{
      max-width: 64ch;
      margin: 14px 0 0;
      color: var(--muted);
    }}
    .artifact-nav {{
      border-radius: 18px;
      padding: 18px 22px;
      margin-bottom: 18px;
    }}
    .artifact-nav ul {{
      list-style: none;
      margin: 12px 0 0;
      padding: 0;
      display: grid;
      gap: 10px;
    }}
    .artifact-nav li {{
      display: flex;
      justify-content: space-between;
      gap: 12px;
      padding-bottom: 10px;
      border-bottom: 1px solid var(--border);
    }}
    .artifact-nav li:last-child {{
      border-bottom: 0;
      padding-bottom: 0;
    }}
    .artifact-nav a {{
      color: inherit;
      text-decoration: none;
      font-weight: 600;
    }}
    .artifact-nav span {{
      color: var(--muted);
      font-size: 0.92rem;
      text-align: right;
    }}
    .artifact-stack {{
      display: grid;
      gap: 18px;
    }}
    .artifact-section {{
      border-radius: 18px;
      padding: 22px;
      overflow: hidden;
    }}
    .artifact-section header {{
      margin-bottom: 14px;
    }}
    .artifact-section header p,
    .artifact-note,
    .artifact-links {{
      margin: 6px 0 0;
      color: var(--muted);
    }}
    .artifact-links a {{
      color: var(--accent);
      font-weight: 600;
      text-decoration: none;
    }}
    .artifact-code {{
      margin: 16px 0 0;
      padding: 18px;
      border-radius: 14px;
      overflow-x: auto;
      border: 1px solid rgba(36, 25, 19, 0.08);
      background: #1d1a18;
      color: #f6efe8;
      font-family: var(--font-code);
      font-size: 0.92rem;
      line-height: 1.55;
    }}
    .artifact-image {{
      margin: 16px 0 0;
      border-radius: 14px;
      overflow: hidden;
      border: 1px solid rgba(36, 25, 19, 0.08);
      background: var(--surface-strong);
    }}
    .artifact-image img {{
      display: block;
      max-width: 100%;
      height: auto;
    }}
    @media (max-width: 720px) {{
      main {{
        width: min(100vw - 20px, 1080px);
        padding-top: 20px;
      }}
      .hero,
      .artifact-section,
      .artifact-nav {{
        padding-left: 18px;
        padding-right: 18px;
      }}
      .artifact-nav li {{
        display: block;
      }}
      .artifact-nav span {{
        display: block;
        text-align: left;
        margin-top: 3px;
      }}
    }}
  </style>
</head>
<body>
  <main>
    <div class="hero-shell">
      <section class="hero">
        <p class="eyebrow">alter0 artifact preview</p>
        <h1>{html.escape(title)}</h1>
        {description_html}
        <div class="hero-meta">
          <span>{len(artifact_args)} artifact(s)</span>
          <span>session-scoped static preview</span>
        </div>
      </section>
    </div>
    <nav class="artifact-nav" aria-label="Artifact list">
      <h2>Included Artifacts</h2>
      <ul>
        {"".join(nav_items)}
      </ul>
    </nav>
    <div class="artifact-stack">
      {"".join(sections)}
    </div>
  </main>
</body>
</html>
"""

(site_dir / "index.html").write_text(document, encoding="utf-8")
PY

DEPLOY_OUTPUT="$(
  bash "${REPO_PATH}/scripts/deploy_test_service.sh" \
    "${SESSION_ID}" \
    "${SERVICE_NAME}" \
    --service-type http \
    --workdir "${SITE_DIR}" \
    --health-path "${HEALTH_PATH}" \
    --command 'exec python3 -m http.server "${PORT}" --bind 127.0.0.1 --directory .'
)"

python3 - "${DEPLOY_OUTPUT}" "${SITE_DIR}" <<'PY'
import json
import sys

payload_text = sys.argv[1]
site_dir = sys.argv[2]
try:
    payload = json.loads(payload_text)
except json.JSONDecodeError:
    print(payload_text)
    raise SystemExit(0)

payload["site_dir"] = site_dir
payload["index_html"] = site_dir + "/index.html"
print(json.dumps(payload))
PY
