package runtime

import (
	"context"
	"time"

	schedulesvc "alter0/app/core/service/schedule"
	"alter0/app/pkg/scheduler"
)

const (
	defaultScheduleDispatchInterval = time.Second
	defaultScheduleDispatchTimeout  = 5 * time.Second
)

type ScheduleOptions struct {
	Enabled          bool
	DispatchInterval time.Duration
	DispatchTimeout  time.Duration
	RunOnStart       bool
	JobName          string
}

func DefaultScheduleOptions() ScheduleOptions {
	return ScheduleOptions{
		Enabled:          true,
		DispatchInterval: defaultScheduleDispatchInterval,
		DispatchTimeout:  defaultScheduleDispatchTimeout,
		RunOnStart:       true,
		JobName:          "schedule-dispatch-due",
	}
}

func RegisterScheduleJobs(jobScheduler *scheduler.Scheduler, scheduleService *schedulesvc.Service, options ScheduleOptions) error {
	if jobScheduler == nil || scheduleService == nil {
		return nil
	}
	opts := sanitizeScheduleOptions(options)
	if !opts.Enabled {
		return nil
	}
	return jobScheduler.Register(scheduler.JobSpec{
		Name:       opts.JobName,
		Interval:   opts.DispatchInterval,
		Timeout:    opts.DispatchTimeout,
		RunOnStart: opts.RunOnStart,
		Run: func(ctx context.Context) error {
			scheduleService.DispatchDue(ctx)
			return nil
		},
	})
}

func sanitizeScheduleOptions(options ScheduleOptions) ScheduleOptions {
	defaults := DefaultScheduleOptions()
	if !options.Enabled && options.DispatchInterval == 0 && options.DispatchTimeout == 0 && !options.RunOnStart && options.JobName == "" {
		options.Enabled = defaults.Enabled
	}
	if options.DispatchInterval <= 0 {
		options.DispatchInterval = defaults.DispatchInterval
	}
	if options.DispatchTimeout <= 0 {
		options.DispatchTimeout = defaults.DispatchTimeout
	}
	if options.JobName == "" {
		options.JobName = defaults.JobName
	}
	// Default behavior is run immediately on startup.
	if !options.RunOnStart {
		options.RunOnStart = defaults.RunOnStart
	}
	return options
}
