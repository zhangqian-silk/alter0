package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryUntilSuccess(t *testing.T) {
	q := New(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	var attempts atomic.Int32
	done := make(chan struct{}, 1)

	_, err := q.Enqueue(Job{
		MaxRetries: 2,
		Run: func(context.Context) error {
			n := attempts.Add(1)
			if n < 3 {
				return errors.New("transient")
			}
			done <- struct{}{}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("expected job to eventually succeed")
	}

	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestAttemptTimeoutCancelsJob(t *testing.T) {
	q := New(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	finished := make(chan error, 1)

	_, err := q.Enqueue(Job{
		AttemptTimeout: 20 * time.Millisecond,
		Run: func(runCtx context.Context) error {
			<-runCtx.Done()
			finished <- runCtx.Err()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	select {
	case err := <-finished:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("expected timeout cancellation")
	}
}

func TestMaxRetriesExhausted(t *testing.T) {
	q := New(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	var attempts atomic.Int32

	_, err := q.Enqueue(Job{
		MaxRetries: 1,
		Run: func(context.Context) error {
			attempts.Add(1)
			return errors.New("always fail")
		},
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	time.Sleep(120 * time.Millisecond)
	if got := attempts.Load(); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestEnqueueContextReturnsWhenQueueIsFull(t *testing.T) {
	q := New(1)

	if _, err := q.Enqueue(Job{Run: func(context.Context) error { return nil }}); err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := q.EnqueueContext(ctx, Job{Run: func(context.Context) error { return nil }})
	if err == nil {
		t.Fatal("expected enqueue timeout error")
	}
	if !errors.Is(err, ErrEnqueueCanceled) {
		t.Fatalf("expected ErrEnqueueCanceled, got %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
