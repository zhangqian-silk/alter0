package domain

import (
	"testing"
	"time"
)

func TestJobValidateIntervalMode(t *testing.T) {
	job := Job{
		ID:       "job-a",
		Name:     "job-a",
		Interval: 5 * time.Minute,
		Enabled:  true,
		Content:  "/time",
	}
	if err := job.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestJobValidateCronMode(t *testing.T) {
	job := Job{
		ID:             "job-b",
		Name:           "job-b",
		ScheduleMode:   ScheduleModeDaily,
		CronExpression: "30 9 * * *",
		Timezone:       "Asia/Shanghai",
		Enabled:        true,
		Content:        "/time",
	}
	if err := job.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestJobValidateRejectsInvalidCronMode(t *testing.T) {
	job := Job{
		ID:           "job-c",
		Name:         "job-c",
		ScheduleMode: ScheduleModeWeekly,
		Timezone:     "UTC",
		Enabled:      true,
		Content:      "/time",
	}
	if err := job.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}
