package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type TaskConfig struct {
	Input      string `json:"input"`
	RetryLimit int    `json:"retry_limit,omitempty"`
}

type Job struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Interval       time.Duration     `json:"-"`
	Enabled        bool              `json:"enabled"`
	SessionID      string            `json:"session_id,omitempty"`
	UserID         string            `json:"user_id,omitempty"`
	ChannelID      string            `json:"channel_id,omitempty"`
	Content        string            `json:"content,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ScheduleMode   ScheduleMode      `json:"schedule_mode,omitempty"`
	Timezone       string            `json:"timezone,omitempty"`
	CronExpression string            `json:"cron_expression"`
	TaskConfig     TaskConfig        `json:"task_config,omitempty"`
}

func (j Job) Validate() error {
	_, err := j.Normalize()
	return err
}

func (j Job) Normalize() (Job, error) {
	normalized := Job{
		ID:             strings.TrimSpace(j.ID),
		Name:           strings.TrimSpace(j.Name),
		Interval:       j.Interval,
		Enabled:        j.Enabled,
		SessionID:      strings.TrimSpace(j.SessionID),
		UserID:         strings.TrimSpace(j.UserID),
		ChannelID:      strings.TrimSpace(j.ChannelID),
		Content:        strings.TrimSpace(j.Content),
		Metadata:       cloneMap(j.Metadata),
		ScheduleMode:   ScheduleMode(strings.ToLower(strings.TrimSpace(string(j.ScheduleMode)))),
		Timezone:       strings.TrimSpace(j.Timezone),
		CronExpression: strings.TrimSpace(j.CronExpression),
		TaskConfig: TaskConfig{
			Input:      strings.TrimSpace(j.TaskConfig.Input),
			RetryLimit: j.TaskConfig.RetryLimit,
		},
	}

	if normalized.TaskConfig.RetryLimit < 0 {
		return Job{}, errors.New("task_config.retry_limit must be non-negative")
	}
	if strings.TrimSpace(j.ID) == "" {
		return Job{}, errors.New("job id is required")
	}
	if normalized.Name == "" {
		normalized.Name = normalized.ID
	}

	if normalized.TaskConfig.Input == "" {
		normalized.TaskConfig.Input = normalized.Content
	}
	if normalized.Content == "" {
		normalized.Content = normalized.TaskConfig.Input
	}
	if normalized.TaskConfig.Input == "" {
		return Job{}, errors.New("task_config.input is required")
	}

	timezone, _, err := NormalizeTimezone(normalized.Timezone)
	if err != nil {
		return Job{}, err
	}
	normalized.Timezone = timezone

	if normalized.CronExpression == "" {
		if normalized.Interval <= 0 {
			return Job{}, errors.New("cron_expression is required")
		}
		if spec, ok := legacyIntervalSpec(normalized.Interval); ok {
			expr, buildErr := BuildCronExpression(spec)
			if buildErr != nil {
				return Job{}, buildErr
			}
			normalized.CronExpression = expr
			normalized.ScheduleMode = spec.Mode
		}
		return normalized, nil
	}

	normalized.CronExpression = canonicalCronExpression(normalized.CronExpression)
	spec, err := ParseCronExpression(normalized.CronExpression)
	if err != nil {
		return Job{}, err
	}
	if normalized.ScheduleMode != "" && normalized.ScheduleMode != spec.Mode {
		return Job{}, fmt.Errorf("schedule_mode %q does not match cron_expression", normalized.ScheduleMode)
	}
	normalized.ScheduleMode = spec.Mode
	if normalized.Interval <= 0 {
		if interval, ok := everyInterval(spec); ok {
			normalized.Interval = interval
		}
	}
	return normalized, nil
}

func (j Job) NextRun(after time.Time) (time.Time, error) {
	normalized, err := j.Normalize()
	if err != nil {
		return time.Time{}, err
	}

	if normalized.CronExpression == "" && normalized.Interval > 0 {
		if after.IsZero() {
			after = time.Now().UTC()
		}
		return after.UTC().Add(normalized.Interval), nil
	}

	spec, err := ParseCronExpression(normalized.CronExpression)
	if err != nil {
		return time.Time{}, err
	}
	_, location, err := NormalizeTimezone(normalized.Timezone)
	if err != nil {
		return time.Time{}, err
	}
	return spec.Next(after, location)
}

func canonicalCronExpression(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	return strings.Join(parts, " ")
}

func legacyIntervalSpec(interval time.Duration) (ScheduleSpec, bool) {
	if interval <= 0 {
		return ScheduleSpec{}, false
	}
	if interval%time.Minute != 0 {
		return ScheduleSpec{}, false
	}

	minutes := int(interval / time.Minute)
	if minutes <= 0 {
		return ScheduleSpec{}, false
	}
	if minutes%1440 == 0 {
		return ScheduleSpec{
			Mode:      ScheduleModeEvery,
			Every:     minutes / 1440,
			EveryUnit: EveryUnitDay,
		}, true
	}
	if minutes%60 == 0 {
		return ScheduleSpec{
			Mode:      ScheduleModeEvery,
			Every:     minutes / 60,
			EveryUnit: EveryUnitHour,
		}, true
	}
	return ScheduleSpec{
		Mode:      ScheduleModeEvery,
		Every:     minutes,
		EveryUnit: EveryUnitMinute,
	}, true
}

func everyInterval(spec ScheduleSpec) (time.Duration, bool) {
	if spec.Mode != ScheduleModeEvery || spec.Every <= 0 {
		return 0, false
	}
	switch spec.EveryUnit {
	case EveryUnitMinute:
		return time.Duration(spec.Every) * time.Minute, true
	case EveryUnitHour:
		return time.Duration(spec.Every) * time.Hour, true
	case EveryUnitDay:
		return time.Duration(spec.Every) * 24 * time.Hour, true
	default:
		return 0, false
	}
}

func cloneMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}
