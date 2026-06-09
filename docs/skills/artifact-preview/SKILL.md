---
name: artifact-preview
description: Publish text, image, and code artifacts as static preview pages on session-scoped subdomains by staging files in the workspace and deploying them through the shared alter0 gateway.
---

# Artifact Preview

- Use this skill when the deliverable is a previewable artifact bundle rather than a full application or routed backend service.
- Use this skill for every static artifact that is intended to be shown to the user in a browser, including generated HTML pages, markdown previews, screenshots, image sets, text reports, JSON examples, code samples, and packaged review artifacts.
- Materialize the content as files in the current workspace first. This flow is designed for text, images, code snippets, markdown, JSON, and similar static assets.
- Publish with `bash docs/skills/artifact-preview/scripts/publish_preview_artifact.sh <session_id> <service_name> --artifact <path> ...`.
- The script builds a static index page under `.alter0/preview-artifacts/<service_name>/site/` and deploys it to `https://<service_name>-<short_hash>.alter0.cn`.
- Keep artifact preview hosts certificate-safe by staying on single-label subdomains covered by `*.alter0.cn`; do not use nested hosts such as `https://<service>.<short_hash>.alter0.cn`.
- Reuse stable `service_name` values so follow-up deploys refresh the same URL.
- If the task needs a full-stack web preview, a routed HTTP service, or the root short-hash host, use `deploy-test-service` instead.
- The alter0 runtime runs on a standalone server. Never report server-local artifact links such as `/srv/...`, `.alter0/workspaces/...`, `file://...`, `localhost`, or `127.0.0.1` as if the user can open them.
- Do not finish a task with a local HTML/file path as the primary artifact. Publish it first, then return the deployed `https://*.alter0.cn` URL.
- If publishing fails, state the deployment blocker and keep the local file path as an implementation detail only, not as the user-facing deliverable link.

## Workflow

1. Write or collect the preview files inside the current workspace.
2. Pick a short service name such as `spec`, `gallery`, `code-review`, or `artifact-demo`.
3. Run the helper script with one or more `--artifact` arguments.
4. Confirm the helper produced an `https://<service_name>-<short_hash>.alter0.cn` URL.
5. Return the deployed preview URL together with the list of included artifacts.

## Command Pattern

```bash
bash docs/skills/artifact-preview/scripts/publish_preview_artifact.sh \
  "$SESSION_ID" docs-preview \
  --title "API Draft Preview" \
  --artifact docs/api-draft.md \
  --artifact screenshots/home.png \
  --artifact internal/interfaces/web/server.go
```

## Rules

- Keep the preview static. Do not hand-roll ad-hoc web servers beyond the helper script's managed `python3 -m http.server` flow.
- Prefer artifact-specific subdomains over overwriting the default `web` host.
- For large codebases, include only the files that matter for review instead of dumping the entire repository into one preview page.
- When the preview contains generated user-facing content, ensure the deployed files match the final answer you report back.
- User-facing links must point to the deployed preview host, not to source files, build output files, temporary directories, or workspace storage paths.
- Generated standalone HTML deliverables must be included with `--artifact` and published through this flow unless they require a full application service, in which case use `deploy-test-service`.
- Screenshots or images referenced by a user-facing report must either be included in the preview bundle or be served from the same deployed preview/service host.
