package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
	taskdomain "alter0/internal/task/domain"
)

type stubTaskOrchestrator struct {
	mu      sync.Mutex
	handler func(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
	calls   []shareddomain.UnifiedMessage
}

func (s *stubTaskOrchestrator) Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	s.mu.Lock()
	s.calls = append(s.calls, msg)
	handler := s.handler
	s.mu.Unlock()

	if handler == nil {
		return shareddomain.OrchestrationResult{
			MessageID: msg.MessageID,
			SessionID: msg.SessionID,
			Route:     shareddomain.RouteNL,
			Output:    "ok",
		}, nil
	}
	return handler(ctx, msg)
}

type stubTaskRecorder struct {
	mu      sync.Mutex
	records []sessiondomain.MessageRecord
}

func (s *stubTaskRecorder) Append(records ...sessiondomain.MessageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, records...)
	return nil
}

func (s *stubTaskRecorder) list() []sessiondomain.MessageRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]sessiondomain.MessageRecord, 0, len(s.records))
	out = append(out, s.records...)
	return out
}

type taskTestIDGenerator struct {
	mu   sync.Mutex
	next int
}

func (g *taskTestIDGenerator) NewID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return "id-" + strconv.Itoa(g.next)
}

func TestServiceShouldRunAsyncByRule(t *testing.T) {
	svc, err := NewService(
		context.Background(),
		&stubTaskOrchestrator{},
		nil,
		&taskTestIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		Options{LongContentThreshold: 8},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	forced := testTaskMessage("s-1", "short", map[string]string{metadataAsyncModeKey: "force"})
	if !svc.ShouldRunAsync(forced) {
		t.Fatalf("expected force mode to run async")
	}
	disabled := testTaskMessage("s-1", "this is very long", map[string]string{metadataAsyncModeKey: "sync"})
	if svc.ShouldRunAsync(disabled) {
		t.Fatalf("expected sync mode to disable async")
	}
	artifact := testTaskMessage("s-1", "short", map[string]string{metadataTaskArtifactKey: "true"})
	if !svc.ShouldRunAsync(artifact) {
		t.Fatalf("expected artifact flag to run async")
	}
	longText := testTaskMessage("s-1", "long-content", nil)
	if !svc.ShouldRunAsync(longText) {
		t.Fatalf("expected long text to run async")
	}
}

func TestServiceSubmitAndCompleteSuccess(t *testing.T) {
	recorder := &stubTaskRecorder{}
	orch := &stubTaskOrchestrator{
		handler: func(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
			return shareddomain.OrchestrationResult{
				MessageID: msg.MessageID,
				SessionID: msg.SessionID,
				Route:     shareddomain.RouteNL,
				Output:    "artifact output",
				Metadata:  map[string]string{"source": "test"},
			}, nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := NewService(
		ctx,
		orch,
		recorder,
		&taskTestIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		Options{WorkerCount: 1, Timeout: 2 * time.Second, MaxRetries: 0},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	queued, err := svc.Submit(testTaskMessage("session-map", "generate artifact report", nil))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if queued.Status != taskdomain.TaskStatusQueued {
		t.Fatalf("expected queued status, got %q", queued.Status)
	}

	completed := waitTaskStatus(t, svc, queued.ID, taskdomain.TaskStatusSuccess, 2*time.Second)
	if completed.Progress != 100 {
		t.Fatalf("expected progress 100, got %d", completed.Progress)
	}
	if len(completed.Artifacts) == 0 {
		t.Fatalf("expected persisted artifact")
	}
	items := svc.ListBySession("session-map")
	if len(items) != 1 || items[0].ID != queued.ID {
		t.Fatalf("expected session 1:N mapping for session-map, got %+v", items)
	}

	waitForRecorderCount(t, recorder, 1, 2*time.Second)
	records := recorder.list()
	if records[0].RouteResult.TaskID != queued.ID {
		t.Fatalf("expected summary route task_id %q, got %q", queued.ID, records[0].RouteResult.TaskID)
	}
}

func TestServiceRetryBeforeSuccess(t *testing.T) {
	var attempts int
	orch := &stubTaskOrchestrator{
		handler: func(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
			attempts++
			if attempts == 1 {
				return shareddomain.OrchestrationResult{
					MessageID: msg.MessageID,
					SessionID: msg.SessionID,
					Route:     shareddomain.RouteNL,
					ErrorCode: "temporary_failure",
				}, errors.New("temporary failure")
			}
			return shareddomain.OrchestrationResult{
				MessageID: msg.MessageID,
				SessionID: msg.SessionID,
				Route:     shareddomain.RouteNL,
				Output:    "recovered",
			}, nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc, err := NewService(
		ctx,
		orch,
		nil,
		&taskTestIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		Options{WorkerCount: 1, Timeout: 2 * time.Second, MaxRetries: 1},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	task, err := svc.Submit(testTaskMessage("s-retry", "generate report", nil))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	completed := waitTaskStatus(t, svc, task.ID, taskdomain.TaskStatusSuccess, 2*time.Second)
	if completed.RetryCount != 1 {
		t.Fatalf("expected retry_count 1, got %d", completed.RetryCount)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestServiceCancelQueuedTask(t *testing.T) {
	blockCh := make(chan struct{})
	orch := &stubTaskOrchestrator{
		handler: func(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
			select {
			case <-ctx.Done():
				return shareddomain.OrchestrationResult{
					MessageID: msg.MessageID,
					SessionID: msg.SessionID,
					Route:     shareddomain.RouteNL,
				}, ctx.Err()
			case <-blockCh:
				return shareddomain.OrchestrationResult{
					MessageID: msg.MessageID,
					SessionID: msg.SessionID,
					Route:     shareddomain.RouteNL,
					Output:    "done",
				}, nil
			}
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc, err := NewService(
		ctx,
		orch,
		nil,
		&taskTestIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		Options{WorkerCount: 1, Timeout: 2 * time.Second, MaxRetries: 0},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	first, err := svc.Submit(testTaskMessage("s-cancel", "first long task", nil))
	if err != nil {
		t.Fatalf("submit first: %v", err)
	}
	_ = waitTaskStatus(t, svc, first.ID, taskdomain.TaskStatusRunning, 2*time.Second)

	second, err := svc.Submit(testTaskMessage("s-cancel", "second queued task", nil))
	if err != nil {
		t.Fatalf("submit second: %v", err)
	}
	canceled, err := svc.Cancel(second.ID)
	if err != nil {
		t.Fatalf("cancel queued task: %v", err)
	}
	if canceled.Status != taskdomain.TaskStatusCanceled {
		t.Fatalf("expected canceled status, got %q", canceled.Status)
	}
	if canceled.ErrorCode != "task_canceled" {
		t.Fatalf("expected task_canceled error code, got %q", canceled.ErrorCode)
	}

	close(blockCh)
	_ = waitTaskStatus(t, svc, first.ID, taskdomain.TaskStatusSuccess, 2*time.Second)
}

func TestServiceTimeoutTransitionsToFailed(t *testing.T) {
	orch := &stubTaskOrchestrator{
		handler: func(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
			<-ctx.Done()
			return shareddomain.OrchestrationResult{
				MessageID: msg.MessageID,
				SessionID: msg.SessionID,
				Route:     shareddomain.RouteNL,
			}, ctx.Err()
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc, err := NewService(
		ctx,
		orch,
		nil,
		&taskTestIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		Options{WorkerCount: 1, Timeout: 80 * time.Millisecond, MaxRetries: 0},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	task, err := svc.Submit(testTaskMessage("s-timeout", "long operation", nil))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	completed := waitTaskStatus(t, svc, task.ID, taskdomain.TaskStatusFailed, 2*time.Second)
	if completed.ErrorCode != "task_timeout" {
		t.Fatalf("expected task_timeout, got %q", completed.ErrorCode)
	}
}

func waitTaskStatus(t *testing.T, svc *Service, taskID string, expected taskdomain.TaskStatus, timeout time.Duration) taskdomain.Task {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		item, ok := svc.Get(taskID)
		if !ok {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if item.Status == expected {
			return item
		}
		time.Sleep(10 * time.Millisecond)
	}
	item, _ := svc.Get(taskID)
	t.Fatalf("task %s not in status %s within %s, got %+v", taskID, expected, timeout, item)
	return taskdomain.Task{}
}

func waitForRecorderCount(t *testing.T, recorder *stubTaskRecorder, expected int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(recorder.list()) >= expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected recorder count >= %d", expected)
}

func testTaskMessage(sessionID string, content string, metadata map[string]string) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:   "message-" + sessionID + "-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		SessionID:   sessionID,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		Metadata:    metadata,
		TraceID:     "trace-" + sessionID,
		ReceivedAt:  time.Now().UTC(),
	}
}
