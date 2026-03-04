package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type cronExpressionSpec struct {
	minute  cronField
	hour    cronField
	day     cronField
	month   cronField
	weekday cronField
}

type cronField struct {
	all    bool
	step   int
	min    int
	max    int
	values map[int]struct{}
}

func ValidateCronExpression(expression string) error {
	_, err := parseCronExpression(expression)
	return err
}

func NextCronFireAt(expression string, location *time.Location, after time.Time) (time.Time, error) {
	spec, err := parseCronExpression(expression)
	if err != nil {
		return time.Time{}, err
	}
	if location == nil {
		location = time.UTC
	}

	base := after.In(location).Truncate(time.Minute).Add(time.Minute)
	limit := base.AddDate(2, 0, 0)
	for cursor := base; !cursor.After(limit); cursor = cursor.Add(time.Minute) {
		if spec.match(cursor) {
			return cursor.UTC(), nil
		}
	}
	return time.Time{}, errors.New("no next fire time found within 2 years")
}

func parseCronExpression(expression string) (cronExpressionSpec, error) {
	fields := strings.Fields(strings.TrimSpace(expression))
	if len(fields) != 5 {
		return cronExpressionSpec{}, errors.New("cron expression must have 5 fields")
	}

	minute, err := parseCronField(fields[0], 0, 59, true, nil)
	if err != nil {
		return cronExpressionSpec{}, fmt.Errorf("invalid minute field: %w", err)
	}
	hour, err := parseCronField(fields[1], 0, 23, true, nil)
	if err != nil {
		return cronExpressionSpec{}, fmt.Errorf("invalid hour field: %w", err)
	}
	day, err := parseCronField(fields[2], 1, 31, true, nil)
	if err != nil {
		return cronExpressionSpec{}, fmt.Errorf("invalid day field: %w", err)
	}
	month, err := parseCronField(fields[3], 1, 12, true, nil)
	if err != nil {
		return cronExpressionSpec{}, fmt.Errorf("invalid month field: %w", err)
	}
	weekday, err := parseCronField(fields[4], 0, 6, true, normalizeWeekday)
	if err != nil {
		return cronExpressionSpec{}, fmt.Errorf("invalid weekday field: %w", err)
	}

	return cronExpressionSpec{
		minute:  minute,
		hour:    hour,
		day:     day,
		month:   month,
		weekday: weekday,
	}, nil
}

func parseCronField(
	raw string,
	min int,
	max int,
	allowStep bool,
	normalize func(int) (int, error),
) (cronField, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return cronField{}, errors.New("empty field")
	}
	if value == "*" {
		return cronField{all: true, min: min, max: max}, nil
	}
	if strings.HasPrefix(value, "*/") {
		if !allowStep {
			return cronField{}, errors.New("step not supported")
		}
		stepValue := strings.TrimSpace(strings.TrimPrefix(value, "*/"))
		step, err := strconv.Atoi(stepValue)
		if err != nil || step <= 0 {
			return cronField{}, errors.New("invalid step value")
		}
		return cronField{step: step, min: min, max: max}, nil
	}

	parts := strings.Split(value, ",")
	values := make(map[int]struct{}, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			return cronField{}, errors.New("invalid list item")
		}
		parsed, err := strconv.Atoi(item)
		if err != nil {
			return cronField{}, errors.New("list value must be number")
		}
		if normalize != nil {
			normalized, normalizeErr := normalize(parsed)
			if normalizeErr != nil {
				return cronField{}, normalizeErr
			}
			parsed = normalized
		}
		if parsed < min || parsed > max {
			return cronField{}, fmt.Errorf("value %d out of range [%d,%d]", parsed, min, max)
		}
		values[parsed] = struct{}{}
	}
	if len(values) == 0 {
		return cronField{}, errors.New("field must not be empty")
	}
	return cronField{values: values, min: min, max: max}, nil
}

func (f cronField) match(value int) bool {
	if f.all {
		return true
	}
	if f.step > 0 {
		if value < f.min {
			return false
		}
		return (value-f.min)%f.step == 0
	}
	_, ok := f.values[value]
	return ok
}

func normalizeWeekday(value int) (int, error) {
	if value == 7 {
		return 0, nil
	}
	if value < 0 || value > 6 {
		return 0, errors.New("weekday must be in [0,6] or 7")
	}
	return value, nil
}

func (s cronExpressionSpec) match(ts time.Time) bool {
	return s.minute.match(ts.Minute()) &&
		s.hour.match(ts.Hour()) &&
		s.day.match(ts.Day()) &&
		s.month.match(int(ts.Month())) &&
		s.weekday.match(int(ts.Weekday()))
}
