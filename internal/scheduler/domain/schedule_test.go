package domain

import (
	"testing"
	"time"
)

func TestParseCronExpressionSupportsVisualModes(t *testing.T) {
	cases := []struct {
		name      string
		expr      string
		mode      ScheduleMode
		every     int
		everyUnit EveryUnit
		hour      int
		minute    int
		weekday   time.Weekday
	}{
		{
			name:      "every minutes",
			expr:      "*/15 * * * *",
			mode:      ScheduleModeEvery,
			every:     15,
			everyUnit: EveryUnitMinute,
		},
		{
			name:      "every hours",
			expr:      "0 */2 * * *",
			mode:      ScheduleModeEvery,
			every:     2,
			everyUnit: EveryUnitHour,
		},
		{
			name:      "every days",
			expr:      "0 0 */3 * *",
			mode:      ScheduleModeEvery,
			every:     3,
			everyUnit: EveryUnitDay,
		},
		{
			name:   "daily fixed",
			expr:   "30 9 * * *",
			mode:   ScheduleModeDaily,
			hour:   9,
			minute: 30,
		},
		{
			name:    "weekly fixed",
			expr:    "45 10 * * 1",
			mode:    ScheduleModeWeekly,
			hour:    10,
			minute:  45,
			weekday: time.Monday,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := ParseCronExpression(tc.expr)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if spec.Mode != tc.mode {
				t.Fatalf("expected mode %s, got %s", tc.mode, spec.Mode)
			}
			if spec.Every != tc.every {
				t.Fatalf("expected every %d, got %d", tc.every, spec.Every)
			}
			if spec.EveryUnit != tc.everyUnit {
				t.Fatalf("expected every unit %s, got %s", tc.everyUnit, spec.EveryUnit)
			}
			if spec.Hour != tc.hour {
				t.Fatalf("expected hour %d, got %d", tc.hour, spec.Hour)
			}
			if spec.Minute != tc.minute {
				t.Fatalf("expected minute %d, got %d", tc.minute, spec.Minute)
			}
			if spec.Weekday != tc.weekday {
				t.Fatalf("expected weekday %d, got %d", tc.weekday, spec.Weekday)
			}
		})
	}
}

func TestBuildCronExpressionRoundTrip(t *testing.T) {
	origin := ScheduleSpec{
		Mode:    ScheduleModeWeekly,
		Hour:    11,
		Minute:  20,
		Weekday: time.Friday,
	}
	expr, err := BuildCronExpression(origin)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if expr != "20 11 * * 5" {
		t.Fatalf("unexpected cron expression %q", expr)
	}

	parsed, err := ParseCronExpression(expr)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed != origin {
		t.Fatalf("expected parsed %#v, got %#v", origin, parsed)
	}
}

func TestScheduleSpecNextUsesTimezone(t *testing.T) {
	spec, err := ParseCronExpression("30 9 * * *")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location failed: %v", err)
	}

	from := time.Date(2026, 3, 5, 1, 45, 0, 0, time.UTC)
	next, err := spec.Next(from, location)
	if err != nil {
		t.Fatalf("next failed: %v", err)
	}
	expected := time.Date(2026, 3, 6, 1, 30, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("expected next %s, got %s", expected.Format(time.RFC3339), next.Format(time.RFC3339))
	}
}

func TestJobNormalizeSupportsLegacyInterval(t *testing.T) {
	job := Job{
		ID:       "job-interval",
		Name:     "job-interval",
		Interval: 2 * time.Hour,
		Enabled:  true,
		TaskConfig: TaskConfig{
			Input: "/time",
		},
	}

	normalized, err := job.Normalize()
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if normalized.CronExpression != "0 */2 * * *" {
		t.Fatalf("expected converted cron expression, got %q", normalized.CronExpression)
	}
	if normalized.ScheduleMode != ScheduleModeEvery {
		t.Fatalf("expected schedule mode every, got %s", normalized.ScheduleMode)
	}
}
