package runtime

import (
	"path/filepath"
	"testing"
	"time"

	orctask "alter0/app/core/orchestrator/task"
	"alter0/app/pkg/scheduler"
)

func TestRegisterMaintenanceJobsDisabled(t *testing.T) {
	s := scheduler.New()
	err := RegisterMaintenanceJobs(s, nil, MaintenanceOptions{Enabled: false})
	if err != nil {
		t.Fatalf("register should not fail: %v", err)
	}
	if got := s.Health().RegisteredJobs; got != 0 {
		t.Fatalf("expected no registered jobs, got %d", got)
	}
}

func TestRegisterMaintenanceJobsWithDefaults(t *testing.T) {
	s := scheduler.New()
	db, err := orctask.NewSQLiteDB(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatalf("new sqlite db failed: %v", err)
	}
	defer db.Close()
	taskStore := orctask.NewStore(db)

	if err := RegisterMaintenanceJobs(s, taskStore, MaintenanceOptions{}); err != nil {
		t.Fatalf("register maintenance jobs: %v", err)
	}
	if got := s.Health().RegisteredJobs; got != 1 {
		t.Fatalf("expected 1 registered job, got %d", got)
	}

	snap := s.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("expected snapshot with 1 job, got %d", len(snap))
	}
	if snap[0].Name != "task-memory-prune" {
		t.Fatalf("unexpected job name: %s", snap[0].Name)
	}
}

func TestSanitizeMaintenanceOptions(t *testing.T) {
	got := sanitizeMaintenanceOptions(MaintenanceOptions{})
	if !got.Enabled {
		t.Fatalf("expected default enabled")
	}
	if got.TaskMemoryRetentionDays != 30 {
		t.Fatalf("unexpected retention days: %d", got.TaskMemoryRetentionDays)
	}
	if got.TaskMemoryOpenRetentionDays != 0 {
		t.Fatalf("unexpected open retention days: %d", got.TaskMemoryOpenRetentionDays)
	}
	if got.PruneInterval != 6*time.Hour {
		t.Fatalf("unexpected interval: %s", got.PruneInterval)
	}
	if got.PruneTimeout != 20*time.Second {
		t.Fatalf("unexpected timeout: %s", got.PruneTimeout)
	}
}
