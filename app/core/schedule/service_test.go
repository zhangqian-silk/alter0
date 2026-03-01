package schedule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"alter0/app/core/scheduler"

	_ "modernc.org/sqlite"
)

type fakeDispatcher struct {
	mu         sync.Mutex
	direct     int
	agentTurn  int
	failDirect bool
}

func (d *fakeDispatcher) DeliverDirect(ctx context.Context, channelID string, to string, content string, meta map[string]interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.direct++
	if d.failDirect {
		return errors.New("direct failed")
	}
	return nil
}

func (d *fakeDispatcher) DeliverAgentTurn(ctx context.Context, channelID string, to string, content string, agentID string, meta map[string]interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.agentTurn++
	return nil
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:schedule-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

func TestServiceExecutesAtJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := openTestStore(t)
	dispatcher := &fakeDispatcher{}
	svc := NewService(store, dispatcher)
	svc.tick = 10 * time.Millisecond

	job, err := svc.Create(ctx, CreateRequest{
		Kind: KindAt,
		At:   time.Now().UTC().Add(20 * time.Millisecond).Format(time.RFC3339),
		Payload: DeliveryPayload{
			Mode:      ModeDirect,
			ChannelID: "cli",
			To:        "user-1",
			Content:   "hello",
		},
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	svc.Start(ctx)
	deadline := time.Now().Add(2 * time.Second)
	for {
		current, err := svc.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("get schedule: %v", err)
		}
		if current.Status == StatusCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected completed status, got %s", current.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}

	runs, err := svc.Runs(ctx, job.ID, 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) == 0 {
		t.Fatalf("expected run records")
	}

	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	if dispatcher.direct == 0 {
		t.Fatalf("expected direct delivery")
	}
}

func TestServiceDispatchWithSchedulerLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := openTestStore(t)
	dispatcher := &fakeDispatcher{}
	svc := NewService(store, dispatcher)

	job, err := svc.Create(ctx, CreateRequest{
		Kind: KindAt,
		At:   time.Now().UTC().Add(20 * time.Millisecond).Format(time.RFC3339),
		Payload: DeliveryPayload{
			Mode:      ModeDirect,
			ChannelID: "cli",
			To:        "user-1",
			Content:   "hello",
		},
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	jobScheduler := scheduler.New()
	if err := jobScheduler.Register(scheduler.JobSpec{
		Name:     "schedule.dispatch_due",
		Interval: 10 * time.Millisecond,
		Run: func(runCtx context.Context) error {
			svc.DispatchDue(runCtx)
			return nil
		},
	}); err != nil {
		t.Fatalf("register scheduler job: %v", err)
	}
	if err := jobScheduler.Start(ctx); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer func() {
		_ = jobScheduler.Stop(time.Second)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		current, err := svc.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("get schedule: %v", err)
		}
		if current.Status == StatusCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected completed status, got %s", current.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}

	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	if dispatcher.direct == 0 {
		t.Fatalf("expected direct delivery via scheduler loop")
	}
}

func TestServiceRetriesAndFailsAtJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := openTestStore(t)
	dispatcher := &fakeDispatcher{failDirect: true}
	svc := NewService(store, dispatcher)
	svc.tick = 10 * time.Millisecond

	job, err := svc.Create(ctx, CreateRequest{
		Kind:          KindAt,
		At:            time.Now().UTC().Format(time.RFC3339),
		MaxRetries:    1,
		RetryDelaySec: 1,
		Payload: DeliveryPayload{
			Mode:      ModeDirect,
			ChannelID: "cli",
			To:        "user-1",
			Content:   "hello",
		},
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	svc.Start(ctx)
	deadline := time.Now().Add(4 * time.Second)
	for {
		current, err := svc.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("get schedule: %v", err)
		}
		if current.Status == StatusFailed {
			if current.Attempt != 2 {
				t.Fatalf("expected attempt=2, got %d", current.Attempt)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected failed status, got %s", current.Status)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestCreateWithIdempotencyKeyReturnsExisting(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	svc := NewService(store, &fakeDispatcher{})

	req := CreateRequest{
		Kind:           KindAt,
		At:             time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
		IdempotencyKey: "same-key",
		Payload: DeliveryPayload{
			Mode:      ModeDirect,
			ChannelID: "cli",
			To:        "user-1",
			Content:   "hello",
		},
	}
	first, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same id, got %s vs %s", first.ID, second.ID)
	}
}

func TestServiceRecoversDueAtJobAfterRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "schedule-restart.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer conn.Close()

	store, err := NewStore(conn)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()
	dispatcher := &fakeDispatcher{}
	svc := NewService(store, dispatcher)
	svc.tick = 10 * time.Millisecond

	job, err := svc.Create(ctx, CreateRequest{
		Kind: KindAt,
		At:   time.Now().UTC().Add(-time.Second).Format(time.RFC3339),
		Payload: DeliveryPayload{
			Mode:      ModeDirect,
			ChannelID: "cli",
			To:        "user-1",
			Content:   "restart-recovery",
		},
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	restartCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	restarted := NewService(store, dispatcher)
	restarted.tick = 10 * time.Millisecond
	restarted.Start(restartCtx)

	deadline := time.Now().Add(2 * time.Second)
	for {
		current, err := restarted.Get(ctx, job.ID)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "locked") {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			t.Fatalf("get schedule after restart: %v", err)
		}
		if current.Status == StatusCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected completed status after restart, got %s", current.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}

	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	if dispatcher.direct == 0 {
		t.Fatalf("expected resumed service to dispatch due schedule")
	}
}
