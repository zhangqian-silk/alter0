package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
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

type JobStatus struct {
	Name         string
	Runs         int64
	LastStartAt  time.Time
	LastEndAt    time.Time
	LastError    string
	LastDuration time.Duration
}

type Scheduler struct {
	mu      sync.Mutex
	jobs    map[string]JobSpec
	status  map[string]JobStatus
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
	jobStop map[string]context.CancelFunc
	wg      sync.WaitGroup
}

func New() *Scheduler {
	return &Scheduler{
		jobs:    make(map[string]JobSpec),
		status:  make(map[string]JobStatus),
		jobStop: make(map[string]context.CancelFunc),
	}
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
	s.status[job.Name] = JobStatus{Name: job.Name}
	if s.started {
		s.startJobLocked(job)
	}
	return nil
}

func (s *Scheduler) Unregister(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[name]; !exists {
		return fmt.Errorf("%w: %s", ErrJobNotFound, name)
	}
	delete(s.jobs, name)
	delete(s.status, name)
	if stop, exists := s.jobStop[name]; exists {
		stop()
		delete(s.jobStop, name)
	}
	return nil
}

func (s *Scheduler) Start(parent context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return ErrSchedulerStart
	}
	ctx, cancel := context.WithCancel(parent)
	s.ctx = ctx
	s.cancel = cancel
	s.started = true
	jobs := make([]JobSpec, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	s.mu.Unlock()

	for _, job := range jobs {
		s.mu.Lock()
		s.startJobLocked(job)
		s.mu.Unlock()
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
	s.ctx = nil
	s.cancel = nil
	s.started = false
	s.jobStop = make(map[string]context.CancelFunc)
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

func (s *Scheduler) Snapshot() []JobStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]JobStatus, 0, len(s.status))
	for _, st := range s.status {
		items = append(items, st)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func (s *Scheduler) startJobLocked(job JobSpec) {
	if !s.started || s.ctx == nil {
		return
	}
	if _, exists := s.jobStop[job.Name]; exists {
		return
	}
	jobCtx, stop := context.WithCancel(s.ctx)
	s.jobStop[job.Name] = stop
	s.wg.Add(1)
	go s.runLoop(jobCtx, job)
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
	start := time.Now()
	s.markJobStart(job.Name, start)

	runCtx := parent
	cancel := func() {}
	if job.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(parent, job.Timeout)
	}
	defer cancel()

	err := job.Run(runCtx)
	end := time.Now()
	s.markJobEnd(job.Name, end, end.Sub(start), err)

	if err != nil {
		log.Printf("[Scheduler] job=%s failed: %v", job.Name, err)
	}
}

func (s *Scheduler) markJobStart(name string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.status[name]
	if !ok {
		st = JobStatus{Name: name}
	}
	st.LastStartAt = at
	s.status[name] = st
}

func (s *Scheduler) markJobEnd(name string, at time.Time, duration time.Duration, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.status[name]
	if !ok {
		st = JobStatus{Name: name}
	}
	st.Runs++
	st.LastEndAt = at
	st.LastDuration = duration
	if err != nil {
		st.LastError = err.Error()
	} else {
		st.LastError = ""
	}
	s.status[name] = st
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
