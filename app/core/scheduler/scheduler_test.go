package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegisterValidation(t *testing.T) {
	s := New()
	if err := s.Register(JobSpec{}); err == nil {
		t.Fatal("expected validation error")
	}

	valid := JobSpec{
		Name:     "tick",
		Interval: 10 * time.Millisecond,
		Run:      func(context.Context) error { return nil },
	}
	if err := s.Register(valid); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := s.Register(valid); !errors.Is(err, ErrJobExists) {
		t.Fatalf("expected ErrJobExists, got: %v", err)
	}
}

func TestStartAndStop(t *testing.T) {
	s := New()
	var runs atomic.Int32

	err := s.Register(JobSpec{
		Name:       "counter",
		Interval:   10 * time.Millisecond,
		RunOnStart: true,
		Run: func(context.Context) error {
			runs.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if runs.Load() == 0 {
		t.Fatal("expected job to run immediately when RunOnStart is true")
	}

	if err := s.Stop(200 * time.Millisecond); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
}

func TestRunOnStartDefaultFalse(t *testing.T) {
	s := New()
	fired := make(chan struct{}, 1)

	err := s.Register(JobSpec{
		Name:     "lazy-start",
		Interval: 50 * time.Millisecond,
		Run: func(context.Context) error {
			select {
			case fired <- struct{}{}:
			default:
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer s.Stop(200 * time.Millisecond)

	select {
	case <-fired:
		t.Fatal("did not expect immediate run when RunOnStart is false")
	case <-time.After(15 * time.Millisecond):
	}
}

func TestJobTimeout(t *testing.T) {
	s := New()
	finished := make(chan struct{}, 1)

	err := s.Register(JobSpec{
		Name:     "timeout",
		Interval: 10 * time.Millisecond,
		Timeout:  20 * time.Millisecond,
		Run: func(ctx context.Context) error {
			<-ctx.Done()
			finished <- struct{}{}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer s.Stop(200 * time.Millisecond)

	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected timeout to cancel job context")
	}
}

func TestRegisterAfterStartRunsJob(t *testing.T) {
	s := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer s.Stop(200 * time.Millisecond)

	runs := make(chan struct{}, 1)
	err := s.Register(JobSpec{
		Name:       "late-job",
		Interval:   20 * time.Millisecond,
		RunOnStart: true,
		Run: func(context.Context) error {
			select {
			case runs <- struct{}{}:
			default:
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	select {
	case <-runs:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected dynamically registered job to run")
	}
}

func TestUnregisterStopsJob(t *testing.T) {
	s := New()
	runs := make(chan struct{}, 8)
	err := s.Register(JobSpec{
		Name:     "removable",
		Interval: 10 * time.Millisecond,
		Run: func(context.Context) error {
			select {
			case runs <- struct{}{}:
			default:
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer s.Stop(200 * time.Millisecond)

	select {
	case <-runs:
	case <-time.After(80 * time.Millisecond):
		t.Fatal("expected initial scheduler run")
	}

	if err := s.Unregister("removable"); err != nil {
		t.Fatalf("unregister failed: %v", err)
	}

	time.Sleep(40 * time.Millisecond)
	for {
		select {
		case <-runs:
			t.Fatal("expected no runs after unregister")
		default:
			return
		}
	}
}

func TestSnapshotTracksLastRunState(t *testing.T) {
	s := New()
	failed := errors.New("boom")

	err := s.Register(JobSpec{
		Name:       "status-job",
		Interval:   100 * time.Millisecond,
		RunOnStart: true,
		Run: func(context.Context) error {
			return failed
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer s.Stop(200 * time.Millisecond)

	deadline := time.Now().Add(150 * time.Millisecond)
	for {
		snap := s.Snapshot()
		if len(snap) == 1 && snap[0].Runs > 0 {
			if snap[0].Name != "status-job" {
				t.Fatalf("unexpected job name: %s", snap[0].Name)
			}
			if snap[0].LastError != failed.Error() {
				t.Fatalf("unexpected last error: %s", snap[0].LastError)
			}
			if snap[0].LastStartAt.IsZero() || snap[0].LastEndAt.IsZero() {
				t.Fatal("expected start and end time to be set")
			}
			if snap[0].LastDuration <= 0 {
				t.Fatal("expected positive run duration")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("snapshot did not observe job run: %+v", snap)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestHealthReflectsLifecycle(t *testing.T) {
	s := New()
	err := s.Register(JobSpec{
		Name:     "health-job",
		Interval: 50 * time.Millisecond,
		Run:      func(context.Context) error { return nil },
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	pre := s.Health()
	if pre.Started {
		t.Fatal("expected scheduler stopped before start")
	}
	if pre.RegisteredJobs != 1 {
		t.Fatalf("unexpected registered jobs: %d", pre.RegisteredJobs)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	post := s.Health()
	if !post.Started {
		t.Fatal("expected started=true after start")
	}
	if post.StartedAt.IsZero() {
		t.Fatal("expected started_at to be set")
	}
	if post.RunningJobs != 1 {
		t.Fatalf("unexpected running jobs: %d", post.RunningJobs)
	}

	if err := s.Stop(200 * time.Millisecond); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	stopped := s.Health()
	if stopped.Started {
		t.Fatal("expected started=false after stop")
	}
}
