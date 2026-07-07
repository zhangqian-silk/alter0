---
name: preview-publish
description: Deploy or refresh session-scoped previews and test services on the shared alter0 gateway, including static artifact previews, frontend builds, and additional HTTP services routed by short-hash hostnames.
---

# Preview Publish

- Use this skill when the task needs a user-openable preview URL, static artifact preview, live web preview, or separately routed test service without editing Nginx.
- Default host: `https://<short_hash>.alter0.cn` for service `web`.
- Additional hosts: `https://<service>-<short_hash>.alter0.cn`.
- Static artifact previews use the same gateway and should be published with `docs/skills/preview-publish/scripts/publish_preview_artifact.sh`.

## Static Artifact Preview

- Use this path for text, image, code, markdown, JSON, screenshots, standalone HTML pages, and packaged review artifacts that need a browser-viewable URL.
- Materialize the content as files in the current workspace first.
- Publish with `bash docs/skills/preview-publish/scripts/publish_preview_artifact.sh <session_id> <service_name> --artifact <path> ...`.
- The helper builds a static index page under `.alter0/preview-artifacts/<service_name>/site/` and deploys it to `https://<service_name>-<short_hash>.alter0.cn`.
- Reuse stable `service_name` values so follow-up deploys refresh the same URL.
- Do not use reserved service names `web` or `travel` for generic artifact previews.

## Tool Contract

- Use `deploy_test_service` for frontend build previews, routed backends, existing upstreams, and full-stack deployments.
- Default `web` deploys should register `service_type=http`, start the current session backend on a local port, and let that backend serve both the latest frontend build and `/api/*`.
- `service_type=frontend_dist` is the static-only fallback. It serves a built `internal/interfaces/web/static/dist` and leaves `/api/*` on the shared runtime backend.
- `service_type=http` registers either an existing `upstream_url` or a `start_command` that boots a service in the session workspace.

## Deployment Rules

- Keep deployments inside the current session namespace. Reuse the current short hash instead of inventing custom domains.
- Prefer stable `service_name` values such as `web`, `api`, `docs`, or `storybook` so repeated deploys update the same routed host.
- Keep additional services on certificate-safe single-label subdomains under `*.alter0.cn`. Do not generate nested hosts such as `https://<service>.<short_hash>.alter0.cn` or `https://<short_hash>.travel.alter0.cn`.
- For `start_command`, assume the deployer injects `PORT` and performs a health probe before registration.
- The standard `scripts/deploy_test_service.sh <session_id>` path now defaults `web` to a full-stack preview. Use `--service-type frontend_dist` only when a static UI-only preview is intentional.
- For frontend work, keep the preview build aligned with the current session repository workspace rather than a stale source checkout.
- For public travel guides, deploy `service_name=travel` on `https://travel-<short_hash>.alter0.cn`. If the session workspace root already contains `index.html`, publish that root directly as the static artifact source.
- Never report server-local artifact links such as `/srv/...`, runtime-root `workspaces/...`, `file://...`, `localhost`, or `127.0.0.1` as user-openable deliverables.
- Do not finish a task with a local HTML/file path as the primary artifact. Publish it first, then return the deployed `https://*.alter0.cn` URL.
