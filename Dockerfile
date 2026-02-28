# syntax=docker/dockerfile:1.7

FROM golang:1.25.7 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/alter0 .

FROM debian:bookworm-slim
WORKDIR /opt/alter0

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/alter0 /usr/local/bin/alter0
COPY config/config.json ./config/config.json

RUN mkdir -p output/db output/logs output/audit output/orchestrator

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/alter0"]
