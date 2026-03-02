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
			ID:        "job1",
			Name:      "job1",
			Interval:  30 * time.Second,
			Enabled:   true,
			SessionID: "cron-session",
			ChannelID: "scheduler-default",
			Content:   "/time",
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
	if jobs[0].Interval != 30*time.Second {
		t.Fatalf("expected 30s interval, got %s", jobs[0].Interval)
	}
}

func TestSchedulerStoreMarkdownRoundTrip(t *testing.T) {
	store := NewSchedulerStore(t.TempDir(), FormatMarkdown)
	err := store.Save(context.Background(), []schedulerdomain.Job{
		{
			ID:        "job2",
			Name:      "job2",
			Interval:  time.Minute,
			Enabled:   true,
			SessionID: "cron-session",
			Content:   "/help",
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
	if jobs[0].Interval != time.Minute {
		t.Fatalf("expected 1m interval, got %s", jobs[0].Interval)
	}
}
