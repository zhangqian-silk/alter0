# app/service

`app/service` is the capability layer between `app/core/orchestrator` and infra adapters.

## Scope

- task service
- command service
- schedule service
- queue service
- store service
- executor service
- tools service
- observability service
- runtime lifecycle service

## Dependency Rule

- `app/core/orchestrator` depends on service interfaces only.
- service implementations do not depend on orchestrator packages.
- infra details (sqlite/fs/external adapters) stay outside orchestrator core.
