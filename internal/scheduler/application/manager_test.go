package application

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	schedulerdomain "alter0/internal/scheduler/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubOrchestrator struct {
	messages chan shareddomain.UnifiedMessage
}

func (s *stubOrchestrator) Handle(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	s.messages <- msg
	return shareddomain.OrchestrationResult{
		MessageID: msg.MessageID,
		SessionID: msg.SessionID,
		Route:     shareddomain.RouteNL,
		Output:    "ok",
	}, nil
}

type noopTelemetry struct{}

func (n *noopTelemetry) CountGateway(_ string)     {}
func (n *noopTelemetry) CountRoute(_ string)       {}
func (n *noopTelemetry) CountCommand(_ string)     {}
func (n *noopTelemetry) CountError(_ string)       {}
func (n *noopTelemetry) CountMemoryEvent(_ string) {}
func (n *noopTelemetry) ObserveDuration(_ string, _ time.Duration) {
}

type atomicIDGenerator struct {
	seq atomic.Int64
}

func (g *atomicIDGenerator) NewID() string {
	return fmt.Sprintf("id-%d", g.seq.Add(1))
}

func TestManagerTriggersCronMessage(t *testing.T) {
	orchestrator := &stubOrchestrator{
		messages: make(chan shareddomain.UnifiedMessage, 4),
	}
	manager := NewManager(
		orchestrator,
		&noopTelemetry{},
		&atomicIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)

	err := manager.Upsert(schedulerdomain.Job{
		ID:        "job-a",
		Name:      "job-a",
		Interval:  20 * time.Millisecond,
		Enabled:   true,
		SessionID: "cron-session",
		ChannelID: "scheduler-default",
		Content:   "/time",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case msg := <-orchestrator.messages:
		if msg.TriggerType != shareddomain.TriggerTypeCron {
			t.Fatalf("expected cron trigger, got %s", msg.TriggerType)
		}
		if msg.ChannelType != shareddomain.ChannelTypeScheduler {
			t.Fatalf("expected scheduler channel type, got %s", msg.ChannelType)
		}
		if msg.CorrelationID != "job-a" {
			t.Fatalf("expected correlation id job-a, got %s", msg.CorrelationID)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("expected cron trigger message but timed out")
	}
}
