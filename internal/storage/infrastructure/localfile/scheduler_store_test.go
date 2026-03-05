package localfile

import (
	"context"
	"testing"
	"time"

	schedulerdomain "alter0/internal/scheduler/domain"
)

func TestSchedulerStoreJSONRoundTrip(t *testing.T) {
	store := NewSchedulerStore(t.TempDir(), FormatJSON)
	err := store.Save(context.Background(), []schedulerdomain.Job{
		{
			ID:             "job1",
			Name:           "daily-report",
			Enabled:        true,
			Timezone:       "UTC",
			ScheduleMode:   schedulerdomain.ScheduleModeDaily,
			CronExpression: "30 9 * * *",
			TaskConfig: schedulerdomain.TaskConfig{
				Input:      "/time",
				RetryLimit: 2,
			},
		},
	})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	jobs, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].CronExpression != "30 9 * * *" {
		t.Fatalf("expected cron expression 30 9 * * *, got %q", jobs[0].CronExpression)
	}
	if jobs[0].TaskConfig.RetryLimit != 2 {
		t.Fatalf("expected retry limit 2, got %d", jobs[0].TaskConfig.RetryLimit)
	}
}

func TestSchedulerStoreMarkdownRoundTrip(t *testing.T) {
	store := NewSchedulerStore(t.TempDir(), FormatMarkdown)
	err := store.Save(context.Background(), []schedulerdomain.Job{
		{
			ID:             "job2",
			Name:           "job2",
			Interval:       time.Minute,
			Enabled:        true,
			Timezone:       "UTC",
			ScheduleMode:   schedulerdomain.ScheduleModeEvery,
			CronExpression: "*/1 * * * *",
			TaskConfig: schedulerdomain.TaskConfig{
				Input: "/help",
			},
		},
	})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	jobs, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].TaskConfig.Input != "/help" {
		t.Fatalf("expected input /help, got %q", jobs[0].TaskConfig.Input)
	}
	if jobs[0].Interval != time.Minute {
		t.Fatalf("expected interval 1m, got %s", jobs[0].Interval)
	}
}
