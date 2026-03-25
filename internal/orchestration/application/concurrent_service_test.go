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

	shareddomain "alter0/internal/shared/domain"
)

type stubOrchestrator struct {
	handleFn func(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

func (s *stubOrchestrator) Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	return s.handleFn(ctx, msg)
}

type queueTelemetrySpy struct {
	*spyTelemetry

	mu           sync.Mutex
	queueEvents  map[string]int
	queueWaitCnt int
	queueDepth   int
	inFlight     int
}

func newQueueTelemetrySpy() *queueTelemetrySpy {
	return &queueTelemetrySpy{
		spyTelemetry: newSpyTelemetry(),
		queueEvents:  map[string]int{},
	}
}

func (t *queueTelemetrySpy) CountQueueEvent(event string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.queueEvents[event]++
}

func (t *queueTelemetrySpy) ObserveQueueWait(_ time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.queueWaitCnt++
}

func (t *queueTelemetrySpy) SetQueueDepth(depth int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.queueDepth = depth
}

func (t *queueTelemetrySpy) SetWorkerInFlight(inFlight int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inFlight = inFlight
}

func TestConcurrentServiceMultiSessionConcurrentAndSessionSerial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan string, 8)
	releaseS1First := make(chan struct{})

	var mu sync.Mutex
	activeBySession := map[string]int{}
	currentActive := 0
	maxActive := 0
	sessionOrder := map[string][]string{}
	sessionOverlap := false

	downstream := &stubOrchestrator{handleFn: func(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
		mu.Lock()
		activeBySession[msg.SessionID]++
		if activeBySession[msg.SessionID] > 1 {
			sessionOverlap = true
		}
		currentActive++
		if currentActive > maxActive {
			maxActive = currentActive
		}
		sessionOrder[msg.SessionID] = append(sessionOrder[msg.SessionID], msg.Content)
		mu.Unlock()

		started <- msg.Content
		if msg.Content == "s1-1" {
			<-releaseS1First
		}
		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		activeBySession[msg.SessionID]--
		currentActive--
		mu.Unlock()

		return shareddomain.OrchestrationResult{Output: msg.Content}, nil
	}}

	svc := NewConcurrentService(
		ctx,
		downstream,
		newQueueTelemetrySpy(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConcurrencyOptions{WorkerCount: 2, MaxQueueSize: 16, QueueTimeout: time.Second},
	)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		_, _ = svc.Handle(context.Background(), testMessage("s1", "s1-1"))
	}()

	if got := waitString(t, started); got != "s1-1" {
		t.Fatalf("expected first started message s1-1, got %q", got)
	}

	go func() {
		defer wg.Done()
		_, _ = svc.Handle(context.Background(), testMessage("s1", "s1-2"))
	}()
	go func() {
		defer wg.Done()
		_, _ = svc.Handle(context.Background(), testMessage("s2", "s2-1"))
	}()

	if got := waitString(t, started); got != "s2-1" {
		t.Fatalf("expected different session to start while s1-1 running, got %q", got)
	}

	select {
	case got := <-started:
		t.Fatalf("unexpected message started before release: %s", got)
	case <-time.After(80 * time.Millisecond):
	}

	close(releaseS1First)
	if got := waitString(t, started); got != "s1-2" {
		t.Fatalf("expected s1-2 to start after s1-1 finishes, got %q", got)
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if sessionOverlap {
		t.Fatalf("same session requests overlapped")
	}
	if maxActive < 2 {
		t.Fatalf("expected multi-session concurrency, max active=%d", maxActive)
	}
	if len(sessionOrder["s1"]) != 2 || sessionOrder["s1"][0] != "s1-1" || sessionOrder["s1"][1] != "s1-2" {
		t.Fatalf("unexpected session order: %#v", sessionOrder["s1"])
	}
}

func TestConcurrentServiceWorkerPoolQueueWaitMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{}, 1)
	downstream := &stubOrchestrator{handleFn: func(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
		if msg.Content == "first" {
			started <- struct{}{}
		}
		time.Sleep(120 * time.Millisecond)
		return shareddomain.OrchestrationResult{Output: msg.Content}, nil
	}}

	svc := NewConcurrentService(
		ctx,
		downstream,
		newQueueTelemetrySpy(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConcurrencyOptions{WorkerCount: 1, MaxQueueSize: 16, QueueTimeout: time.Second},
	)

	firstDone := make(chan struct{})
	go func() {
		_, _ = svc.Handle(context.Background(), testMessage("s1", "first"))
		close(firstDone)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first request did not start")
	}

	result, err := svc.Handle(context.Background(), testMessage("s2", "second"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	waitMs := mustMetadataInt(t, result.Metadata, "queue_wait_ms")
	if waitMs < 100 {
		t.Fatalf("expected queue wait >=100ms, got %d", waitMs)
	}
	if result.Metadata["timeout"] != "false" {
		t.Fatalf("expected timeout metadata false, got %q", result.Metadata["timeout"])
	}

	<-firstDone
}

func TestConcurrentServiceQueueTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	started := make(chan struct{}, 1)
	telemetry := newQueueTelemetrySpy()
	downstream := &stubOrchestrator{handleFn: func(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
		if msg.Content == "hold" {
			started <- struct{}{}
			<-release
		}
		return shareddomain.OrchestrationResult{Output: msg.Content}, nil
	}}

	svc := NewConcurrentService(
		ctx,
		downstream,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConcurrencyOptions{WorkerCount: 1, MaxQueueSize: 8, QueueTimeout: 80 * time.Millisecond},
	)

	go func() {
		_, _ = svc.Handle(context.Background(), testMessage("s1", "hold"))
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("hold request did not start")
	}

	result, err := svc.Handle(context.Background(), testMessage("s2", "queued"))
	if !errors.Is(err, ErrQueueTimeout) {
		t.Fatalf("expected ErrQueueTimeout, got %v", err)
	}
	if result.ErrorCode != ErrorCodeQueueTimeout {
		t.Fatalf("expected error code %s, got %s", ErrorCodeQueueTimeout, result.ErrorCode)
	}
	if result.Metadata["timeout"] != "true" {
		t.Fatalf("expected timeout metadata true, got %q", result.Metadata["timeout"])
	}
	if mustMetadataInt(t, result.Metadata, "queue_wait_ms") < 80 {
		t.Fatalf("queue_wait_ms should be >= timeout")
	}

	telemetry.mu.Lock()
	timeoutEvents := telemetry.queueEvents["timeout"]
	telemetry.mu.Unlock()
	if timeoutEvents == 0 {
		t.Fatalf("expected timeout queue event")
	}

	close(release)
}

func TestConcurrentServiceSameSessionWaitDoesNotTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	started := make(chan struct{}, 1)
	downstream := &stubOrchestrator{handleFn: func(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
		if msg.Content == "hold" {
			started <- struct{}{}
			<-release
		}
		return shareddomain.OrchestrationResult{Output: msg.Content}, nil
	}}

	svc := NewConcurrentService(
		ctx,
		downstream,
		newQueueTelemetrySpy(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConcurrencyOptions{WorkerCount: 1, MaxQueueSize: 8, QueueTimeout: 40 * time.Millisecond},
	)

	go func() {
		_, _ = svc.Handle(context.Background(), testMessage("s1", "hold"))
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("hold request did not start")
	}

	resultCh := make(chan shareddomain.OrchestrationResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := svc.Handle(context.Background(), testMessage("s1", "follow-up"))
		resultCh <- result
		errCh <- err
	}()

	time.Sleep(100 * time.Millisecond)
	close(release)

	var result shareddomain.OrchestrationResult
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("follow-up request did not return")
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("follow-up error missing")
	}
	if result.ErrorCode != "" {
		t.Fatalf("expected no error code, got %q", result.ErrorCode)
	}
	if mustMetadataInt(t, result.Metadata, "queue_wait_ms") < 100 {
		t.Fatalf("expected same-session wait to be preserved")
	}
}

func TestConcurrentServiceQueueCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	started := make(chan struct{}, 1)
	var execCount int64
	var execMu sync.Mutex
	downstream := &stubOrchestrator{handleFn: func(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
		execMu.Lock()
		execCount++
		execMu.Unlock()
		if msg.Content == "hold" {
			started <- struct{}{}
			<-release
		}
		return shareddomain.OrchestrationResult{Output: msg.Content}, nil
	}}

	svc := NewConcurrentService(
		ctx,
		downstream,
		newQueueTelemetrySpy(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConcurrencyOptions{WorkerCount: 1, MaxQueueSize: 8, QueueTimeout: time.Second},
	)

	go func() {
		_, _ = svc.Handle(context.Background(), testMessage("s1", "hold"))
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("hold request did not start")
	}

	queueCtx, queueCancel := context.WithCancel(context.Background())
	resultCh := make(chan shareddomain.OrchestrationResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := svc.Handle(queueCtx, testMessage("s2", "cancel-me"))
		resultCh <- result
		errCh <- err
	}()

	time.Sleep(30 * time.Millisecond)
	queueCancel()

	var result shareddomain.OrchestrationResult
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("queue cancel did not return")
	}
	var err error
	select {
	case err = <-errCh:
	case <-time.After(time.Second):
		t.Fatal("queue cancel error missing")
	}
	if !errors.Is(err, ErrQueueCanceled) {
		t.Fatalf("expected ErrQueueCanceled, got %v", err)
	}
	if result.ErrorCode != ErrorCodeQueueCanceled {
		t.Fatalf("expected error code %s, got %s", ErrorCodeQueueCanceled, result.ErrorCode)
	}
	if result.Metadata["timeout"] != "false" {
		t.Fatalf("expected timeout=false metadata, got %q", result.Metadata["timeout"])
	}

	close(release)
	time.Sleep(20 * time.Millisecond)

	execMu.Lock()
	count := execCount
	execMu.Unlock()
	if count != 1 {
		t.Fatalf("expected only first request executed, got %d", count)
	}
}

func TestConcurrentServiceRejectWhenOverloaded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	started := make(chan struct{}, 1)
	downstream := &stubOrchestrator{handleFn: func(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
		if msg.Content == "hold" {
			started <- struct{}{}
			<-release
		}
		return shareddomain.OrchestrationResult{Output: msg.Content}, nil
	}}

	telemetry := newQueueTelemetrySpy()
	svc := NewConcurrentService(
		ctx,
		downstream,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConcurrencyOptions{WorkerCount: 1, MaxQueueSize: 1, QueueTimeout: time.Second},
	)

	go func() {
		_, _ = svc.Handle(context.Background(), testMessage("s1", "hold"))
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("hold request did not start")
	}

	queueCtx, queueCancel := context.WithCancel(context.Background())
	defer queueCancel()
	go func() {
		_, _ = svc.Handle(queueCtx, testMessage("s2", "queued"))
	}()
	waitForQueueDepth(t, telemetry, 1)

	result, err := svc.Handle(context.Background(), testMessage("s3", "rejected"))
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
	if result.ErrorCode != ErrorCodeRateLimited {
		t.Fatalf("expected error code %s, got %s", ErrorCodeRateLimited, result.ErrorCode)
	}
	if result.Metadata["degraded"] != "true" {
		t.Fatalf("expected degraded metadata true, got %q", result.Metadata["degraded"])
	}

	telemetry.mu.Lock()
	rejectedEvents := telemetry.queueEvents["rejected"]
	telemetry.mu.Unlock()
	if rejectedEvents == 0 {
		t.Fatalf("expected rejected queue event")
	}

	queueCancel()
	close(release)
}

func testMessage(sessionID, content string) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:   "msg-" + content,
		SessionID:   sessionID,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		TraceID:     "trace-" + content,
		ReceivedAt:  time.Now().UTC(),
	}
}

func waitString(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for start signal")
		return ""
	}
}

func mustMetadataInt(t *testing.T, metadata map[string]string, key string) int {
	t.Helper()
	value, ok := metadata[key]
	if !ok {
		t.Fatalf("missing metadata %s", key)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("invalid metadata %s=%q: %v", key, value, err)
	}
	return parsed
}

func waitForQueueDepth(t *testing.T, telemetry *queueTelemetrySpy, expected int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		telemetry.mu.Lock()
		depth := telemetry.queueDepth
		telemetry.mu.Unlock()
		if depth == expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("queue depth did not reach %d", expected)
}
