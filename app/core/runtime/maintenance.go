package runtime

import (
	"context"
	"log"
	"time"

	"alter0/app/core/orchestrator/task"
	"alter0/app/core/scheduler"
)

const defaultTaskMemoryRetentionDays = 30

type MaintenanceOptions struct {
	Enabled                 bool
	TaskMemoryRetentionDays int
	PruneInterval           time.Duration
	PruneTimeout            time.Duration
}

func DefaultMaintenanceOptions() MaintenanceOptions {
	return MaintenanceOptions{
		Enabled:                 true,
		TaskMemoryRetentionDays: defaultTaskMemoryRetentionDays,
		PruneInterval:           6 * time.Hour,
		PruneTimeout:            20 * time.Second,
	}
}

func RegisterMaintenanceJobs(jobScheduler *scheduler.Scheduler, taskStore *task.Store, options MaintenanceOptions) error {
	if jobScheduler == nil || taskStore == nil {
		return nil
	}
	opts := sanitizeMaintenanceOptions(options)
	if !opts.Enabled {
		return nil
	}
	return jobScheduler.Register(scheduler.JobSpec{
		Name:       "task-memory-prune",
		Interval:   opts.PruneInterval,
		Timeout:    opts.PruneTimeout,
		RunOnStart: true,
		Run: func(ctx context.Context) error {
			cutoff := time.Now().Add(-time.Duration(opts.TaskMemoryRetentionDays) * 24 * time.Hour).Unix()
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

func sanitizeMaintenanceOptions(options MaintenanceOptions) MaintenanceOptions {
	defaults := DefaultMaintenanceOptions()
	if !options.Enabled && options.PruneInterval == 0 && options.PruneTimeout == 0 && options.TaskMemoryRetentionDays == 0 {
		options.Enabled = defaults.Enabled
	}
	if options.PruneInterval <= 0 {
		options.PruneInterval = defaults.PruneInterval
	}
	if options.PruneTimeout <= 0 {
		options.PruneTimeout = defaults.PruneTimeout
	}
	if options.TaskMemoryRetentionDays <= 0 {
		options.TaskMemoryRetentionDays = defaults.TaskMemoryRetentionDays
	}
	return options
}
