package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrQueueStarted    = errors.New("queue: already started")
	ErrQueueStopped    = errors.New("queue: stopped")
	ErrEnqueueCanceled = errors.New("queue: enqueue canceled")
)

type Job struct {
	ID             string
	MaxRetries     int
	RetryDelay     time.Duration
	AttemptTimeout time.Duration
	Run            func(context.Context) error
}

type Queue struct {
	mu        sync.Mutex
	jobs      chan queuedJob
	started   bool
	stopping  bool
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	nextID    atomic.Uint64
	inFlight  atomic.Int64
	enqueued  atomic.Uint64
	completed atomic.Uint64
	failed    atomic.Uint64
	retried   atomic.Uint64
}

type queuedJob struct {
	job     Job
	attempt int
}

type Stats struct {
	Started   bool   `json:"started"`
	Depth     int    `json:"depth"`
	Capacity  int    `json:"capacity"`
	Enqueued  uint64 `json:"enqueued"`
	Completed uint64 `json:"completed"`
	Failed    uint64 `json:"failed"`
	Retried   uint64 `json:"retried"`
}

type ShutdownReport struct {
	PendingAtStart  int           `json:"pending_at_start"`
	InFlightAtStart int64         `json:"in_flight_at_start"`
	DrainedJobs     uint64        `json:"drained_jobs"`
	TimedOut        bool          `json:"timed_out"`
	RemainingDepth  int           `json:"remaining_depth"`
	RemainingFlight int64         `json:"remaining_in_flight"`
	Elapsed         time.Duration `json:"elapsed"`
}

func New(buffer int) *Queue {
	if buffer <= 0 {
		buffer = 64
	}
	return &Queue{jobs: make(chan queuedJob, buffer)}
}

func (q *Queue) Enqueue(job Job) (string, error) {
	return q.EnqueueContext(context.Background(), job)
}

func (q *Queue) EnqueueContext(ctx context.Context, job Job) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateJob(job); err != nil {
		return "", err
	}
	if job.ID == "" {
		job.ID = fmt.Sprintf("q-%d", q.nextID.Add(1))
	}

	q.mu.Lock()
	jobs := q.jobs
	stopping := q.stopping
	q.mu.Unlock()
	if stopping {
		return "", ErrQueueStopped
	}

	select {
	case jobs <- queuedJob{job: job, attempt: 0}:
		q.enqueued.Add(1)
		return job.ID, nil
	case <-ctx.Done():
		return "", fmt.Errorf("%w: %w", ErrEnqueueCanceled, ctx.Err())
	}
}

func (q *Queue) Stats() Stats {
	q.mu.Lock()
	started := q.started
	q.mu.Unlock()

	return Stats{
		Started:   started,
		Depth:     len(q.jobs),
		Capacity:  cap(q.jobs),
		Enqueued:  q.enqueued.Load(),
		Completed: q.completed.Load(),
		Failed:    q.failed.Load(),
		Retried:   q.retried.Load(),
	}
}

func (q *Queue) Start(parent context.Context, workers int) error {
	if workers <= 0 {
		workers = 1
	}

	q.mu.Lock()
	if q.started {
		q.mu.Unlock()
		return ErrQueueStarted
	}
	ctx, cancel := context.WithCancel(parent)
	q.cancel = cancel
	q.started = true
	q.stopping = false
	q.mu.Unlock()

	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker(ctx)
	}
	return nil
}

func (q *Queue) Stop(timeout time.Duration) error {
	_, err := q.StopWithReport(timeout)
	return err
}

func (q *Queue) StopWithReport(timeout time.Duration) (ShutdownReport, error) {
	q.mu.Lock()
	if !q.started {
		q.mu.Unlock()
		return ShutdownReport{}, nil
	}
	cancel := q.cancel
	q.cancel = nil
	q.started = false
	q.stopping = true
	report := ShutdownReport{
		PendingAtStart:  len(q.jobs),
		InFlightAtStart: q.inFlight.Load(),
	}
	baseDone := q.completed.Load() + q.failed.Load()
	q.mu.Unlock()

	startedAt := time.Now()
	timedOut := false
	if timeout <= 0 {
		for {
			if len(q.jobs) == 0 && q.inFlight.Load() == 0 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
		q.wg.Wait()
	} else {
		deadline := time.NewTimer(timeout)
		ticker := time.NewTicker(5 * time.Millisecond)
	deferLoop:
		for {
			if len(q.jobs) == 0 && q.inFlight.Load() == 0 {
				cancel()
				done := make(chan struct{})
				go func() {
					defer close(done)
					q.wg.Wait()
				}()
				select {
				case <-done:
				case <-deadline.C:
					timedOut = true
				}
				break deferLoop
			}
			select {
			case <-deadline.C:
				timedOut = true
				cancel()
				break deferLoop
			case <-ticker.C:
			}
		}
		deadline.Stop()
		ticker.Stop()
	}

	report.Elapsed = time.Since(startedAt)
	nowDone := q.completed.Load() + q.failed.Load()
	if nowDone > baseDone {
		report.DrainedJobs = nowDone - baseDone
	}
	report.TimedOut = timedOut
	report.RemainingDepth = len(q.jobs)
	report.RemainingFlight = q.inFlight.Load()

	q.mu.Lock()
	q.stopping = false
	q.mu.Unlock()

	if timedOut {
		return report, fmt.Errorf("queue: stop timeout after %s", timeout)
	}
	return report, nil
}

func (q *Queue) worker(ctx context.Context) {
	defer q.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-q.jobs:
			q.inFlight.Add(1)
			q.runOnce(ctx, item)
			q.inFlight.Add(-1)
		}
	}
}

func (q *Queue) runOnce(parent context.Context, item queuedJob) {
	attempt := item.attempt + 1
	runCtx := parent
	cancel := func() {}
	if item.job.AttemptTimeout > 0 {
		runCtx, cancel = context.WithTimeout(parent, item.job.AttemptTimeout)
	}
	err := item.job.Run(runCtx)
	cancel()
	if err == nil {
		q.completed.Add(1)
		return
	}

	if parent.Err() != nil {
		return
	}

	if attempt >= item.job.MaxRetries+1 {
		q.failed.Add(1)
		return
	}
	q.retried.Add(1)
	if item.job.RetryDelay > 0 {
		timer := time.NewTimer(item.job.RetryDelay)
		defer timer.Stop()
		select {
		case <-parent.Done():
			return
		case <-timer.C:
		}
	}

	select {
	case <-parent.Done():
		return
	case q.jobs <- queuedJob{job: item.job, attempt: attempt}:
	}
}

func validateJob(job Job) error {
	if job.Run == nil {
		return errors.New("queue: job run callback is required")
	}
	if job.MaxRetries < 0 {
		return errors.New("queue: max retries cannot be negative")
	}
	if job.AttemptTimeout < 0 {
		return errors.New("queue: attempt timeout cannot be negative")
	}
	if job.RetryDelay < 0 {
		return errors.New("queue: retry delay cannot be negative")
	}
	return nil
}
