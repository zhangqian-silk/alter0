package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	ErrJobExists      = errors.New("scheduler: job already exists")
	ErrJobNotFound    = errors.New("scheduler: job not found")
	ErrSchedulerStart = errors.New("scheduler: already started")
)

type JobSpec struct {
	Name       string
	Interval   time.Duration
	Timeout    time.Duration
	RunOnStart bool
	Run        func(context.Context) error
}

type Scheduler struct {
	mu      sync.Mutex
	jobs    map[string]JobSpec
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func New() *Scheduler {
	return &Scheduler{jobs: make(map[string]JobSpec)}
}

func (s *Scheduler) Register(job JobSpec) error {
	if err := validateJob(job); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[job.Name]; exists {
		return fmt.Errorf("%w: %s", ErrJobExists, job.Name)
	}
	s.jobs[job.Name] = job
	return nil
}

func (s *Scheduler) Unregister(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[name]; !exists {
		return fmt.Errorf("%w: %s", ErrJobNotFound, name)
	}
	delete(s.jobs, name)
	return nil
}

func (s *Scheduler) Start(parent context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return ErrSchedulerStart
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.started = true
	jobs := make([]JobSpec, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	s.mu.Unlock()

	for _, job := range jobs {
		s.wg.Add(1)
		go s.runLoop(ctx, job)
	}
	return nil
}

func (s *Scheduler) Stop(timeout time.Duration) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.cancel = nil
	s.started = false
	s.mu.Unlock()

	cancel()
	if timeout <= 0 {
		s.wg.Wait()
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.wg.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("scheduler: stop timeout after %s", timeout)
	}
}

func (s *Scheduler) runLoop(ctx context.Context, job JobSpec) {
	defer s.wg.Done()
	if job.RunOnStart {
		s.runOnce(ctx, job)
	}
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx, job)
		}
	}
}

func (s *Scheduler) runOnce(parent context.Context, job JobSpec) {
	runCtx := parent
	cancel := func() {}
	if job.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(parent, job.Timeout)
	}
	defer cancel()

	if err := job.Run(runCtx); err != nil {
		log.Printf("[Scheduler] job=%s failed: %v", job.Name, err)
	}
}

func validateJob(job JobSpec) error {
	if job.Name == "" {
		return errors.New("scheduler: job name is required")
	}
	if job.Interval <= 0 {
		return errors.New("scheduler: job interval must be greater than zero")
	}
	if job.Run == nil {
		return errors.New("scheduler: job run callback is required")
	}
	return nil
}
