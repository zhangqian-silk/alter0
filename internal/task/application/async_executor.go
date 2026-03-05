package application

import (
	"context"
	"strings"
	"sync"
)

const asyncExecutorMaxConcurrency = 5

type asyncExecutor struct {
	workerCount int
	run         func(ctx context.Context, taskID string, workerID int)

	mu      sync.Mutex
	cond    *sync.Cond
	queue   []string
	stopped bool
}

func newAsyncExecutor(workerCount int, run func(ctx context.Context, taskID string, workerID int)) *asyncExecutor {
	if workerCount <= 0 {
		workerCount = asyncExecutorMaxConcurrency
	}
	if workerCount > asyncExecutorMaxConcurrency {
		workerCount = asyncExecutorMaxConcurrency
	}
	executor := &asyncExecutor{
		workerCount: workerCount,
		run:         run,
		queue:       []string{},
	}
	executor.cond = sync.NewCond(&executor.mu)
	return executor
}

func (e *asyncExecutor) start(ctx context.Context) {
	if e == nil || e.run == nil {
		return
	}
	for workerID := 1; workerID <= e.workerCount; workerID++ {
		go e.runWorker(ctx, workerID)
	}
	go func() {
		<-ctx.Done()
		e.mu.Lock()
		e.stopped = true
		e.mu.Unlock()
		e.cond.Broadcast()
	}()
}

func (e *asyncExecutor) enqueue(taskID string) int {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopped {
		return 0
	}
	e.queue = append(e.queue, taskID)
	position := len(e.queue)
	e.cond.Signal()
	return position
}

func (e *asyncExecutor) remove(taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for idx := range e.queue {
		if e.queue[idx] != taskID {
			continue
		}
		e.queue = append(e.queue[:idx], e.queue[idx+1:]...)
		return true
	}
	return false
}

func (e *asyncExecutor) position(taskID string) int {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for idx := range e.queue {
		if e.queue[idx] == taskID {
			return idx + 1
		}
	}
	return 0
}

func (e *asyncExecutor) runWorker(ctx context.Context, workerID int) {
	for {
		taskID, ok := e.next(ctx)
		if !ok {
			return
		}
		e.run(ctx, taskID, workerID)
	}
}

func (e *asyncExecutor) next(ctx context.Context) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for len(e.queue) == 0 && !e.stopped && ctx.Err() == nil {
		e.cond.Wait()
	}
	if e.stopped || ctx.Err() != nil {
		return "", false
	}
	taskID := e.queue[0]
	e.queue = e.queue[1:]
	return taskID, true
}
