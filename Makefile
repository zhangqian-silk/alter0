.PHONY: run build test fmt backup restore
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
