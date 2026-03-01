package runtime

import (
	"context"
	"log"
	"time"

	"alter0/app/service/scheduler"
)

const (
	defaultTaskMemoryRetentionDays     = 30
	defaultTaskMemoryOpenRetentionDays = 0
)

type MaintenanceOptions struct {
	Enabled                     bool
	TaskMemoryRetentionDays     int
	TaskMemoryOpenRetentionDays int
	PruneInterval               time.Duration
	PruneTimeout                time.Duration
}

type TaskMemoryPruner interface {
	PruneTaskMemoryByClosedAt(context.Context, int64) (int64, error)
	PruneTaskMemoryByOpenUpdatedAt(context.Context, int64) (int64, error)
}

func DefaultMaintenanceOptions() MaintenanceOptions {
	return MaintenanceOptions{
		Enabled:                     true,
		TaskMemoryRetentionDays:     defaultTaskMemoryRetentionDays,
		TaskMemoryOpenRetentionDays: defaultTaskMemoryOpenRetentionDays,
		PruneInterval:               6 * time.Hour,
		PruneTimeout:                20 * time.Second,
	}
}

func RegisterMaintenanceJobs(jobScheduler *scheduler.Scheduler, taskStore TaskMemoryPruner, options MaintenanceOptions) error {
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
			now := time.Now()
			closedCutoff := now.Add(-time.Duration(opts.TaskMemoryRetentionDays) * 24 * time.Hour).Unix()
			prunedClosed, err := taskStore.PruneTaskMemoryByClosedAt(ctx, closedCutoff)
			if err != nil {
				return err
			}
			if prunedClosed > 0 {
				log.Printf("[Maintenance] task-memory-prune scope=closed removed=%d cutoff=%d", prunedClosed, closedCutoff)
			}

			if opts.TaskMemoryOpenRetentionDays > 0 {
				openCutoff := now.Add(-time.Duration(opts.TaskMemoryOpenRetentionDays) * 24 * time.Hour).Unix()
				prunedOpen, err := taskStore.PruneTaskMemoryByOpenUpdatedAt(ctx, openCutoff)
				if err != nil {
					return err
				}
				if prunedOpen > 0 {
					log.Printf("[Maintenance] task-memory-prune scope=open removed=%d cutoff=%d", prunedOpen, openCutoff)
				}
			}
			return nil
		},
	})
}

func sanitizeMaintenanceOptions(options MaintenanceOptions) MaintenanceOptions {
	defaults := DefaultMaintenanceOptions()
	if !options.Enabled && options.PruneInterval == 0 && options.PruneTimeout == 0 && options.TaskMemoryRetentionDays == 0 && options.TaskMemoryOpenRetentionDays == 0 {
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
	if options.TaskMemoryOpenRetentionDays < 0 {
		options.TaskMemoryOpenRetentionDays = defaults.TaskMemoryOpenRetentionDays
	}
	return options
}
