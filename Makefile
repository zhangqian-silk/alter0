.PHONY: run build test test-stability check-config-boundary check-service-boundary config-governance cost-threshold-history cost-threshold-reconcile risk-benchmark channel-chaos-drill channel-chaos-candidates channel-chaos-calibration competitor-tracking-refresh fmt backup restore docs-sync-check deploy-check integration-matrix rollback-drill release-gate
.DEFAULT_GOAL := run

run:
	go run .

build:
	go build -o bin/alter0 .

test:
	GOTOOLCHAIN=$${GOTOOLCHAIN:-auto} GOSUMDB=$${GOSUMDB:-sum.golang.org} go test ./... -count=1

test-stability:
	./scripts/check-test-stability.sh

check-config-boundary:
	./scripts/check-config-boundary.sh

check-service-boundary:
	./scripts/check-service-boundary.sh

config-governance:
	./scripts/check-config-governance.sh

cost-threshold-history:
	./scripts/check-cost-threshold-history.sh

cost-threshold-reconcile:
	./scripts/check-cost-threshold-reconcile.sh

risk-benchmark:
	./scripts/check-risk-benchmark.sh

channel-chaos-drill:
	./scripts/check-channel-chaos-drill.sh

channel-chaos-candidates:
	./scripts/check-channel-chaos-candidates.sh

channel-chaos-calibration:
	./scripts/check-channel-chaos-calibration.sh

competitor-tracking-refresh:
	./scripts/update-competitor-tracking.sh

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
