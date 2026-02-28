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

func TestWorkersProcessJobsInParallel(t *testing.T) {
	q := New(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 3); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	const totalJobs = 6
	gate := make(chan struct{})
	var running atomic.Int32
	var peak atomic.Int32
	var completed atomic.Int32

	for i := 0; i < totalJobs; i++ {
		_, err := q.Enqueue(Job{
			Run: func(context.Context) error {
				active := running.Add(1)
				for {
					max := peak.Load()
					if active <= max || peak.CompareAndSwap(max, active) {
						break
					}
				}
				<-gate
				running.Add(-1)
				completed.Add(1)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("enqueue %d failed: %v", i, err)
		}
	}

	deadline := time.Now().Add(300 * time.Millisecond)
	for peak.Load() < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("expected peak parallelism to reach 3 workers, got %d", peak.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}

	close(gate)

	completeDeadline := time.Now().Add(300 * time.Millisecond)
	for completed.Load() < totalJobs {
		if time.Now().After(completeDeadline) {
			t.Fatalf("expected all jobs to complete, completed=%d", completed.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestQueueRecoversAfterJobFailure(t *testing.T) {
	q := New(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	firstRan := make(chan struct{}, 1)
	secondDone := make(chan struct{}, 1)

	_, err := q.Enqueue(Job{
		Run: func(context.Context) error {
			select {
			case firstRan <- struct{}{}:
			default:
			}
			return errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("enqueue first job failed: %v", err)
	}

	_, err = q.Enqueue(Job{
		Run: func(context.Context) error {
			secondDone <- struct{}{}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("enqueue second job failed: %v", err)
	}

	select {
	case <-firstRan:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected first job to run")
	}

	select {
	case <-secondDone:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("expected queue to continue processing after a failed job")
	}

	stats := q.Stats()
	if stats.Failed != 1 {
		t.Fatalf("expected failed=1, got %+v", stats)
	}
	if stats.Completed != 1 {
		t.Fatalf("expected completed=1, got %+v", stats)
	}
}

func TestStopWithReportDrainsQueue(t *testing.T) {
	q := New(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	done := make(chan struct{}, 3)
	for i := 0; i < 3; i++ {
		if _, err := q.Enqueue(Job{Run: func(context.Context) error {
			done <- struct{}{}
			return nil
		}}); err != nil {
			t.Fatalf("enqueue %d failed: %v", i, err)
		}
	}

	report, err := q.StopWithReport(300 * time.Millisecond)
	if err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if report.TimedOut {
		t.Fatalf("expected graceful stop report, got %+v", report)
	}
	if report.DrainedJobs != 3 {
		t.Fatalf("expected drained jobs=3, got %+v", report)
	}
	if report.RemainingDepth != 0 || report.RemainingFlight != 0 {
		t.Fatalf("expected empty queue after stop, got %+v", report)
	}
	if stats := q.Stats(); stats.Workers != 0 {
		t.Fatalf("expected workers=0 after stop, got %+v", stats)
	}

	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("expected all jobs to finish before stop returns")
		}
	}
}

func TestStatsIncludeInFlight(t *testing.T) {
	q := New(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	if _, err := q.Enqueue(Job{Run: func(context.Context) error {
		started <- struct{}{}
		<-release
		return nil
	}}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected job to start")
	}

	stats := q.Stats()
	if stats.Workers != 1 {
		t.Fatalf("expected workers=1, got %+v", stats)
	}
	if stats.InFlight != 1 {
		t.Fatalf("expected in_flight=1, got %+v", stats)
	}

	close(release)
	time.Sleep(30 * time.Millisecond)

	stats = q.Stats()
	if stats.InFlight != 0 {
		t.Fatalf("expected in_flight=0 after completion, got %+v", stats)
	}
}

func TestStopWithReportTimeout(t *testing.T) {
	q := New(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	started := make(chan struct{}, 1)
	if _, err := q.Enqueue(Job{Run: func(context.Context) error {
		started <- struct{}{}
		time.Sleep(120 * time.Millisecond)
		return nil
	}}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected job to start")
	}

	report, err := q.StopWithReport(20 * time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !report.TimedOut {
		t.Fatalf("expected timeout report, got %+v", report)
	}
	if report.InFlightAtStart != 1 {
		t.Fatalf("expected in-flight at stop start to be 1, got %+v", report)
	}
	if report.Elapsed < 20*time.Millisecond {
		t.Fatalf("expected elapsed to include timeout, got %+v", report)
	}

	time.Sleep(150 * time.Millisecond)
}
