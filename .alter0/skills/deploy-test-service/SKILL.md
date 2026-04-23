---
name: deploy-test-service
description: Deploy or refresh a session-scoped test service on the shared alter0 gateway, including frontend preview builds and additional HTTP services routed by short-hash hostnames.
---

# Deploy Test Service

- Use this skill when the task needs a live preview or a separately routed test service without editing Nginx.
- Default host: `https://<short_hash>.alter0.cn` for service `web`.
- Additional hosts: `https://<service>.<short_hash>.alter0.cn`.

## Tool Contract

- Use `deploy_test_service` for deployment.
- Default `web` deploys should register `service_type=http`, start the current session backend on a local port, and let that backend serve both the latest frontend build and `/api/*`.
- `service_type=frontend_dist` is the static-only fallback. It serves a built `internal/interfaces/web/static/dist` and leaves `/api/*` on the shared runtime backend.
- `service_type=http` registers either an existing `upstream_url` or a `start_command` that boots a service in the session workspace.

## Deployment Rules

- Keep deployments inside the current session namespace. Reuse the current short hash instead of inventing custom domains.
- Prefer stable `service_name` values such as `web`, `api`, `docs`, or `storybook` so repeated deploys update the same routed host.
- For `start_command`, assume the deployer injects `PORT` and performs a health probe before registration.
- The standard `scripts/deploy_test_service.sh <session_id>` path now defaults `web` to a full-stack preview. Use `--service-type frontend_dist` only when a static UI-only preview is intentional.
- For frontend work, keep the preview build aligned with the current session repository workspace rather than a stale source checkout.
