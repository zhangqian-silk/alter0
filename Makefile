.PHONY: run build test fmt
.DEFAULT_GOAL := run

run:
	go run .

build:
	go build -o bin/alter0 .

test:
	go test ./...

fmt:
	gofmt -w .
