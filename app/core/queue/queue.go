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
	ErrQueueStarted = errors.New("queue: already started")
)

type Job struct {
	ID             string
	MaxRetries     int
	RetryDelay     time.Duration
	AttemptTimeout time.Duration
	Run            func(context.Context) error
}

type Queue struct {
	mu      sync.Mutex
	jobs    chan queuedJob
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	nextID  atomic.Uint64
}

type queuedJob struct {
	job     Job
	attempt int
}

func New(buffer int) *Queue {
	if buffer <= 0 {
		buffer = 64
	}
	return &Queue{jobs: make(chan queuedJob, buffer)}
}

func (q *Queue) Enqueue(job Job) (string, error) {
	if err := validateJob(job); err != nil {
		return "", err
	}
	if job.ID == "" {
		job.ID = fmt.Sprintf("q-%d", q.nextID.Add(1))
	}

	q.mu.Lock()
	jobs := q.jobs
	q.mu.Unlock()

	jobs <- queuedJob{job: job, attempt: 0}
	return job.ID, nil
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
	q.mu.Unlock()

	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker(ctx)
	}
	return nil
}

func (q *Queue) Stop(timeout time.Duration) error {
	q.mu.Lock()
	if !q.started {
		q.mu.Unlock()
		return nil
	}
	cancel := q.cancel
	q.cancel = nil
	q.started = false
	q.mu.Unlock()

	cancel()
	if timeout <= 0 {
		q.wg.Wait()
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		q.wg.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("queue: stop timeout after %s", timeout)
	}
}

func (q *Queue) worker(ctx context.Context) {
	defer q.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-q.jobs:
			q.runOnce(ctx, item)
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
		return
	}

	if parent.Err() != nil {
		return
	}

	if attempt > item.job.MaxRetries+1 {
		return
	}
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
