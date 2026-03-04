package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubPersistenceDownstream struct {
	result shareddomain.OrchestrationResult
	err    error
}

func (s *stubPersistenceDownstream) Handle(_ context.Context, _ shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	return s.result, s.err
}

type spySessionRecorder struct {
	records []sessiondomain.MessageRecord
	err     error
}

func (s *spySessionRecorder) Append(records ...sessiondomain.MessageRecord) error {
	s.records = append(s.records, records...)
	return s.err
}

type fixedIDGenerator struct {
	nextID string
}

func (g *fixedIDGenerator) NewID() string {
	return g.nextID
}

func TestSessionPersistenceServiceRecordsUserAndAssistantMessages(t *testing.T) {
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-1",
			SessionID: "s-1",
			Route:     shareddomain.RouteNL,
			Output:    "answer",
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-1"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-1",
		SessionID:   "s-1",
		Content:     "question",
		ReceivedAt:  time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-1",
	}

	if _, err := service.Handle(context.Background(), msg); err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	if recorder.records[0].Role != sessiondomain.MessageRoleUser || recorder.records[1].Role != sessiondomain.MessageRoleAssistant {
		t.Fatalf("unexpected roles: %+v", recorder.records)
	}
	if recorder.records[1].MessageID != "assistant-1" {
		t.Fatalf("expected assistant message id assistant-1, got %q", recorder.records[1].MessageID)
	}
	if recorder.records[1].RouteResult.Route != shareddomain.RouteNL {
		t.Fatalf("expected route nl, got %q", recorder.records[1].RouteResult.Route)
	}
}

func TestSessionPersistenceServiceKeepsResultWhenStoreFails(t *testing.T) {
	expectedErr := errors.New("nl failed")
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-1",
			SessionID: "s-1",
			Route:     shareddomain.RouteNL,
			ErrorCode: "nl_execution_failed",
		},
		err: expectedErr,
	}
	recorder := &spySessionRecorder{err: errors.New("disk unavailable")}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-1"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-1",
		SessionID:   "s-1",
		Content:     "question",
		ReceivedAt:  time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-1",
	}

	result, err := service.Handle(context.Background(), msg)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected downstream error %v, got %v", expectedErr, err)
	}
	if result.ErrorCode != "nl_execution_failed" {
		t.Fatalf("expected error_code nl_execution_failed, got %q", result.ErrorCode)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	if recorder.records[1].Content != expectedErr.Error() {
		t.Fatalf("expected assistant content %q, got %q", expectedErr.Error(), recorder.records[1].Content)
	}
}

func TestSessionPersistenceServicePersistsTaskIDFromMetadata(t *testing.T) {
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-task",
			SessionID: "s-task",
			Route:     shareddomain.RouteNL,
			Output:    "done",
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-task"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-task",
		SessionID:   "s-task",
		Content:     "question",
		Metadata:    map[string]string{"task_id": "task-123"},
		ReceivedAt:  time.Date(2026, 3, 3, 13, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-task",
	}

	if _, err := service.Handle(context.Background(), msg); err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	if recorder.records[0].RouteResult.TaskID != "task-123" {
		t.Fatalf("expected user record task id task-123, got %q", recorder.records[0].RouteResult.TaskID)
	}
	if recorder.records[1].RouteResult.TaskID != "task-123" {
		t.Fatalf("expected assistant record task id task-123, got %q", recorder.records[1].RouteResult.TaskID)
	}
}

func TestSessionPersistenceServicePersistsCronSourceMetadata(t *testing.T) {
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-cron",
			SessionID: "cron-session",
			Route:     shareddomain.RouteNL,
			Output:    "ok",
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-cron"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID: "msg-cron",
		SessionID: "cron-session",
		Content:   "cron message",
		Metadata: map[string]string{
			"job_id":   "job-nightly",
			"fired_at": "2026-03-04T09:30:00Z",
		},
		ReceivedAt:  time.Date(2026, 3, 4, 9, 30, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeCron,
		ChannelID:   "scheduler-default",
		ChannelType: shareddomain.ChannelTypeScheduler,
		TraceID:     "trace-cron",
	}

	if _, err := service.Handle(context.Background(), msg); err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	for _, record := range recorder.records {
		if record.TriggerType != shareddomain.TriggerTypeCron {
			t.Fatalf("expected trigger_type cron, got %s", record.TriggerType)
		}
		if record.JobID != "job-nightly" {
			t.Fatalf("expected job_id job-nightly, got %s", record.JobID)
		}
		if record.FiredAt.Format(time.RFC3339) != "2026-03-04T09:30:00Z" {
			t.Fatalf("unexpected fired_at %s", record.FiredAt.Format(time.RFC3339))
		}
	}
}
