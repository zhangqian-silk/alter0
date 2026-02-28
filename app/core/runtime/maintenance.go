package runtime

import (
	"context"
	"log"
	"time"

	"alter0/app/core/orchestrator/task"
	"alter0/app/core/scheduler"
)

const defaultTaskMemoryRetentionDays = 30

func RegisterMaintenanceJobs(jobScheduler *scheduler.Scheduler, taskStore *task.Store) error {
	if jobScheduler == nil || taskStore == nil {
		return nil
	}
	return jobScheduler.Register(scheduler.JobSpec{
		Name:       "task-memory-prune",
		Interval:   6 * time.Hour,
		Timeout:    20 * time.Second,
		RunOnStart: true,
		Run: func(ctx context.Context) error {
			cutoff := time.Now().Add(-defaultTaskMemoryRetentionDays * 24 * time.Hour).Unix()
			pruned, err := taskStore.PruneTaskMemoryByClosedAt(ctx, cutoff)
			if err != nil {
				return err
			}
			if pruned > 0 {
				log.Printf("[Maintenance] task-memory-prune removed=%d cutoff=%d", pruned, cutoff)
			}
			return nil
		},
	})
}
