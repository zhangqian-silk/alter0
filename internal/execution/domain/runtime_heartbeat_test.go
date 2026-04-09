package domain

import (
	"context"
	"testing"
	"time"
)

func TestRuntimeHeartbeatReporterEmitsWithTimestamp(t *testing.T) {
	var got RuntimeHeartbeat
	ctx := WithRuntimeHeartbeatReporter(context.Background(), func(heartbeat RuntimeHeartbeat) {
		got = heartbeat
	})

	EmitRuntimeHeartbeat(ctx, RuntimeHeartbeat{Source: "codex", Message: "running"})

	if got.Source != "codex" || got.Message != "running" {
		t.Fatalf("heartbeat = %+v, want source codex and message running", got)
	}
	if got.Timestamp.IsZero() {
		t.Fatalf("Timestamp is zero, want generated timestamp")
	}
	if got.Timestamp.Location() != time.UTC {
		t.Fatalf("Timestamp location = %v, want UTC", got.Timestamp.Location())
	}
}

func TestRuntimeHeartbeatReporterPreservesExplicitTimestamp(t *testing.T) {
	explicit := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	var got RuntimeHeartbeat
	ctx := WithRuntimeHeartbeatReporter(nil, func(heartbeat RuntimeHeartbeat) {
		got = heartbeat
	})

	EmitRuntimeHeartbeat(ctx, RuntimeHeartbeat{Source: "task", Timestamp: explicit})

	if !got.Timestamp.Equal(explicit) {
		t.Fatalf("Timestamp = %v, want %v", got.Timestamp, explicit)
	}
}

func TestRuntimeHeartbeatReporterIgnoresMissingReporter(t *testing.T) {
	EmitRuntimeHeartbeat(nil, RuntimeHeartbeat{Source: "nil"})
	EmitRuntimeHeartbeat(context.Background(), RuntimeHeartbeat{Source: "missing"})

	ctx := WithRuntimeHeartbeatReporter(context.Background(), nil)
	EmitRuntimeHeartbeat(ctx, RuntimeHeartbeat{Source: "nil-reporter"})
}
