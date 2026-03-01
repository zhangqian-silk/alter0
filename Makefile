.PHONY: run build test fmt backup restore docs-sync-check deploy-check integration-matrix rollback-drill release-gate
.DEFAULT_GOAL := run

run:
	go run .

build:
	go build -o bin/alter0 .

test:
	go test ./...

fmt:
	gofmt -w .

backup:
	./scripts/backup-local.sh

restore:
	@if [ -z "$(BACKUP)" ]; then \
		echo "usage: make restore BACKUP=output/backups/alter0-backup-<timestamp>.tar.gz"; \
		exit 1; \
	fi
	./scripts/restore-local.sh "$(BACKUP)"

docs-sync-check:
	./scripts/check-doc-sync.sh

deploy-check:
	./scripts/check-deploy-assets.sh

integration-matrix:
	./scripts/run-integration-matrix.sh

rollback-drill:
	./scripts/drill-backup-restore.sh

release-gate:
	./scripts/check-release-gates.sh
