.DEFAULT_GOAL := run

.PHONY: run start test help

WEB_ADDR ?= 127.0.0.1:18088

run:
	go run ./cmd/alter0 -web-addr $(WEB_ADDR)

start: run

test:
	go test ./...

help:
	@echo "Targets:"
	@echo "  make        Start alter0 service"
	@echo "  make run    Start alter0 service"
	@echo "  make run WEB_ADDR=127.0.0.1:<your-port>   Start with custom listen addr"
	@echo "  make test   Run all tests"
