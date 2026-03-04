package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ScheduleMode string

const (
	ScheduleModeInterval   ScheduleMode = "interval"
	ScheduleModeDaily      ScheduleMode = "daily"
	ScheduleModeWeekly     ScheduleMode = "weekly"
	ScheduleModeExpression ScheduleMode = "expression"
	defaultTimezone                     = "UTC"
)

type TaskConfig struct {
	RetryStrategy string `json:"retry_strategy,omitempty"`
}

type Job struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Interval       time.Duration     `json:"-"`
	ScheduleMode   ScheduleMode      `json:"schedule_mode,omitempty"`
	CronExpression string            `json:"cron_expression,omitempty"`
	Timezone       string            `json:"timezone,omitempty"`
	Enabled        bool              `json:"enabled"`
	SessionID      string            `json:"session_id,omitempty"`
	UserID         string            `json:"user_id,omitempty"`
	ChannelID      string            `json:"channel_id,omitempty"`
	Content        string            `json:"content"`
	TaskConfig     TaskConfig        `json:"task_config,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func NormalizeScheduleMode(raw string) ScheduleMode {
	switch ScheduleMode(strings.ToLower(strings.TrimSpace(raw))) {
	case ScheduleModeInterval:
		return ScheduleModeInterval
	case ScheduleModeDaily:
		return ScheduleModeDaily
	case ScheduleModeWeekly:
		return ScheduleModeWeekly
	case ScheduleModeExpression:
		return ScheduleModeExpression
	default:
		return ""
	}
}

func (m ScheduleMode) IsValid() bool {
	return m == ScheduleModeInterval ||
		m == ScheduleModeDaily ||
		m == ScheduleModeWeekly ||
		m == ScheduleModeExpression
}

func (j Job) EffectiveScheduleMode() ScheduleMode {
	mode := NormalizeScheduleMode(string(j.ScheduleMode))
	if mode.IsValid() {
		return mode
	}
	if strings.TrimSpace(j.CronExpression) != "" {
		return ScheduleModeExpression
	}
	return ScheduleModeInterval
}

func (j Job) EffectiveTimezone() string {
	timezone := strings.TrimSpace(j.Timezone)
	if timezone == "" {
		return defaultTimezone
	}
	return timezone
}

func (j Job) EffectiveCronExpression() string {
	expression := strings.TrimSpace(j.CronExpression)
	if expression != "" {
		return expression
	}
	if j.Interval <= 0 {
		return ""
	}
	if j.Interval < time.Minute || j.Interval%time.Minute != 0 {
		return ""
	}

	totalMinutes := int(j.Interval / time.Minute)
	if totalMinutes <= 0 {
		return ""
	}
	if totalMinutes < 60 {
		return "*/" + strconv.Itoa(totalMinutes) + " * * * *"
	}
	if totalMinutes%(24*60) == 0 {
		return "0 0 */" + strconv.Itoa(totalMinutes/(24*60)) + " * *"
	}
	if totalMinutes%60 == 0 {
		return "0 */" + strconv.Itoa(totalMinutes/60) + " * * *"
	}
	return ""
}

func (j Job) Normalized() Job {
	normalized := j
	normalized.ID = strings.ToLower(strings.TrimSpace(normalized.ID))
	normalized.Name = strings.TrimSpace(normalized.Name)
	if normalized.Name == "" {
		normalized.Name = normalized.ID
	}
	normalized.ScheduleMode = normalized.EffectiveScheduleMode()
	normalized.Timezone = normalized.EffectiveTimezone()
	if normalized.ScheduleMode != ScheduleModeInterval {
		normalized.CronExpression = normalized.EffectiveCronExpression()
	}
	normalized.TaskConfig.RetryStrategy = strings.ToLower(strings.TrimSpace(normalized.TaskConfig.RetryStrategy))
	return normalized
}

func (j Job) Validate() error {
	if strings.TrimSpace(j.ID) == "" {
		return errors.New("job id is required")
	}
	if strings.TrimSpace(j.Content) == "" {
		return errors.New("content is required")
	}

	mode := j.EffectiveScheduleMode()
	if mode == "" {
		return errors.New("schedule_mode is required")
	}

	switch mode {
	case ScheduleModeInterval:
		if j.Interval <= 0 {
			return errors.New("interval must be greater than zero")
		}
	case ScheduleModeDaily, ScheduleModeWeekly, ScheduleModeExpression:
		expression := j.EffectiveCronExpression()
		if expression == "" {
			return errors.New("cron_expression is required")
		}
		if err := ValidateCronExpression(expression); err != nil {
			return fmt.Errorf("invalid cron_expression: %w", err)
		}
	default:
		return errors.New("schedule_mode must be interval/daily/weekly/expression")
	}

	if _, err := time.LoadLocation(j.EffectiveTimezone()); err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}
	return nil
}
