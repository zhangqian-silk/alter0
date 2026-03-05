package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	maxScheduleSearchDays = 370
)

type ScheduleMode string

const (
	ScheduleModeEvery  ScheduleMode = "every"
	ScheduleModeDaily  ScheduleMode = "daily"
	ScheduleModeWeekly ScheduleMode = "weekly"
)

type EveryUnit string

const (
	EveryUnitMinute EveryUnit = "minute"
	EveryUnitHour   EveryUnit = "hour"
	EveryUnitDay    EveryUnit = "day"
)

type ScheduleSpec struct {
	Mode      ScheduleMode
	Every     int
	EveryUnit EveryUnit
	Hour      int
	Minute    int
	Weekday   time.Weekday
}

func ParseCronExpression(expr string) (ScheduleSpec, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return ScheduleSpec{}, errors.New("cron_expression is required")
	}
	parts := strings.Fields(trimmed)
	if len(parts) != 5 {
		return ScheduleSpec{}, errors.New("cron_expression must include 5 fields")
	}
	minuteField := parts[0]
	hourField := parts[1]
	dayField := parts[2]
	monthField := parts[3]
	weekdayField := parts[4]

	if monthField != "*" {
		return ScheduleSpec{}, errors.New("cron_expression month field must be *")
	}

	if step, ok := parseStep(minuteField); ok && hourField == "*" && dayField == "*" && weekdayField == "*" {
		spec := ScheduleSpec{
			Mode:      ScheduleModeEvery,
			Every:     step,
			EveryUnit: EveryUnitMinute,
		}
		if err := spec.Validate(); err != nil {
			return ScheduleSpec{}, err
		}
		return spec, nil
	}

	if minuteField == "0" {
		if step, ok := parseStep(hourField); ok && dayField == "*" && weekdayField == "*" {
			spec := ScheduleSpec{
				Mode:      ScheduleModeEvery,
				Every:     step,
				EveryUnit: EveryUnitHour,
			}
			if err := spec.Validate(); err != nil {
				return ScheduleSpec{}, err
			}
			return spec, nil
		}
		if hourField == "0" {
			if step, ok := parseStep(dayField); ok && weekdayField == "*" {
				spec := ScheduleSpec{
					Mode:      ScheduleModeEvery,
					Every:     step,
					EveryUnit: EveryUnitDay,
				}
				if err := spec.Validate(); err != nil {
					return ScheduleSpec{}, err
				}
				return spec, nil
			}
		}
	}

	if dayField == "*" && weekdayField == "*" {
		hour, err := parseCronNumber(hourField, 0, 23)
		if err != nil {
			return ScheduleSpec{}, errors.New("cron_expression hour must be 0-23")
		}
		minute, err := parseCronNumber(minuteField, 0, 59)
		if err != nil {
			return ScheduleSpec{}, errors.New("cron_expression minute must be 0-59")
		}
		spec := ScheduleSpec{
			Mode:   ScheduleModeDaily,
			Hour:   hour,
			Minute: minute,
		}
		if err := spec.Validate(); err != nil {
			return ScheduleSpec{}, err
		}
		return spec, nil
	}

	if dayField == "*" {
		hour, err := parseCronNumber(hourField, 0, 23)
		if err != nil {
			return ScheduleSpec{}, errors.New("cron_expression hour must be 0-23")
		}
		minute, err := parseCronNumber(minuteField, 0, 59)
		if err != nil {
			return ScheduleSpec{}, errors.New("cron_expression minute must be 0-59")
		}
		weekday, err := parseCronWeekday(weekdayField)
		if err != nil {
			return ScheduleSpec{}, errors.New("cron_expression weekday must be 0-6 or SUN-SAT")
		}
		spec := ScheduleSpec{
			Mode:    ScheduleModeWeekly,
			Hour:    hour,
			Minute:  minute,
			Weekday: weekday,
		}
		if err := spec.Validate(); err != nil {
			return ScheduleSpec{}, err
		}
		return spec, nil
	}

	return ScheduleSpec{}, errors.New("unsupported cron_expression, use every/daily/weekly pattern")
}

func BuildCronExpression(spec ScheduleSpec) (string, error) {
	if err := spec.Validate(); err != nil {
		return "", err
	}
	switch spec.Mode {
	case ScheduleModeEvery:
		switch spec.EveryUnit {
		case EveryUnitMinute:
			return fmt.Sprintf("*/%d * * * *", spec.Every), nil
		case EveryUnitHour:
			return fmt.Sprintf("0 */%d * * *", spec.Every), nil
		case EveryUnitDay:
			return fmt.Sprintf("0 0 */%d * *", spec.Every), nil
		default:
			return "", errors.New("every_unit must be minute/hour/day")
		}
	case ScheduleModeDaily:
		return fmt.Sprintf("%d %d * * *", spec.Minute, spec.Hour), nil
	case ScheduleModeWeekly:
		return fmt.Sprintf("%d %d * * %d", spec.Minute, spec.Hour, int(spec.Weekday)), nil
	default:
		return "", errors.New("schedule_mode must be every/daily/weekly")
	}
}

func (s ScheduleSpec) Validate() error {
	switch s.Mode {
	case ScheduleModeEvery:
		if s.Every <= 0 {
			return errors.New("every value must be greater than zero")
		}
		switch s.EveryUnit {
		case EveryUnitMinute, EveryUnitHour, EveryUnitDay:
		default:
			return errors.New("every_unit must be minute/hour/day")
		}
		return nil
	case ScheduleModeDaily:
		if s.Hour < 0 || s.Hour > 23 {
			return errors.New("hour must be within 0..23")
		}
		if s.Minute < 0 || s.Minute > 59 {
			return errors.New("minute must be within 0..59")
		}
		return nil
	case ScheduleModeWeekly:
		if s.Hour < 0 || s.Hour > 23 {
			return errors.New("hour must be within 0..23")
		}
		if s.Minute < 0 || s.Minute > 59 {
			return errors.New("minute must be within 0..59")
		}
		if s.Weekday < time.Sunday || s.Weekday > time.Saturday {
			return errors.New("weekday must be within 0..6")
		}
		return nil
	default:
		return errors.New("schedule_mode must be every/daily/weekly")
	}
}

func (s ScheduleSpec) Next(after time.Time, location *time.Location) (time.Time, error) {
	if err := s.Validate(); err != nil {
		return time.Time{}, err
	}
	if location == nil {
		location = time.UTC
	}
	if after.IsZero() {
		after = time.Now().UTC()
	}

	current := after.In(location).Truncate(time.Minute).Add(time.Minute)
	deadline := current.Add(maxScheduleSearchDays * 24 * time.Hour)
	for !current.After(deadline) {
		if s.match(current) {
			return current.UTC(), nil
		}
		current = current.Add(time.Minute)
	}
	return time.Time{}, errors.New("no schedulable fire time within search window")
}

func NormalizeTimezone(raw string) (string, *time.Location, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "UTC", time.UTC, nil
	}
	location, err := time.LoadLocation(trimmed)
	if err != nil {
		return "", nil, errors.New("invalid timezone")
	}
	return location.String(), location, nil
}

func (s ScheduleSpec) match(ts time.Time) bool {
	switch s.Mode {
	case ScheduleModeEvery:
		switch s.EveryUnit {
		case EveryUnitMinute:
			return ts.Minute()%s.Every == 0
		case EveryUnitHour:
			return ts.Minute() == 0 && ts.Hour()%s.Every == 0
		case EveryUnitDay:
			return ts.Minute() == 0 && ts.Hour() == 0 && ((ts.Day()-1)%s.Every == 0)
		default:
			return false
		}
	case ScheduleModeDaily:
		return ts.Hour() == s.Hour && ts.Minute() == s.Minute
	case ScheduleModeWeekly:
		return ts.Weekday() == s.Weekday && ts.Hour() == s.Hour && ts.Minute() == s.Minute
	default:
		return false
	}
}

func parseStep(value string) (int, bool) {
	if !strings.HasPrefix(value, "*/") {
		return 0, false
	}
	number := strings.TrimSpace(strings.TrimPrefix(value, "*/"))
	if number == "" {
		return 0, false
	}
	step, err := strconv.Atoi(number)
	if err != nil || step <= 0 {
		return 0, false
	}
	return step, true
}

func parseCronNumber(raw string, min int, max int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, errors.New("number is empty")
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, err
	}
	if value < min || value > max {
		return 0, errors.New("number out of range")
	}
	return value, nil
}

func parseCronWeekday(raw string) (time.Weekday, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(raw))
	if trimmed == "" {
		return time.Sunday, errors.New("weekday is empty")
	}
	switch trimmed {
	case "0", "7", "SUN":
		return time.Sunday, nil
	case "1", "MON":
		return time.Monday, nil
	case "2", "TUE":
		return time.Tuesday, nil
	case "3", "WED":
		return time.Wednesday, nil
	case "4", "THU":
		return time.Thursday, nil
	case "5", "FRI":
		return time.Friday, nil
	case "6", "SAT":
		return time.Saturday, nil
	default:
		return time.Sunday, errors.New("weekday out of range")
	}
}
