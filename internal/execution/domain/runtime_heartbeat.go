package domain

import (
	"context"
	"time"
)

type RuntimeHeartbeat struct {
	Source    string    `json:"source,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type RuntimeHeartbeatReporter func(RuntimeHeartbeat)

type runtimeHeartbeatReporterKey struct{}

func WithRuntimeHeartbeatReporter(ctx context.Context, reporter RuntimeHeartbeatReporter) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, runtimeHeartbeatReporterKey{}, reporter)
}

func EmitRuntimeHeartbeat(ctx context.Context, heartbeat RuntimeHeartbeat) {
	if ctx == nil {
		return
	}
	reporter, ok := ctx.Value(runtimeHeartbeatReporterKey{}).(RuntimeHeartbeatReporter)
	if !ok || reporter == nil {
		return
	}
	if heartbeat.Timestamp.IsZero() {
		heartbeat.Timestamp = time.Now().UTC()
	}
	reporter(heartbeat)
}
