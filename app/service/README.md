# app/service

`app/service` is the reusable capability layer between `app/core` and infra adapters.

## Scope

- executor service
- queue service
- scheduler service
- schedule service
- store service
- tools service
- observability service

## Core-Owned Business Modules

Business-oriented orchestration logic is owned by `app/core`:

- `app/core/orchestrator/command`
- `app/core/orchestrator/execlog`
- `app/core/orchestrator/task`
- `app/core/runtime` (status + maintenance orchestration)

Legacy counterparts under `app/service` are retained for transition/testing only.

## Dependency Rule

- `app/core` may depend on reusable service capabilities.
- service implementations do not depend on core packages.
- infra details (sqlite/fs/external adapters) stay outside core business modules.
