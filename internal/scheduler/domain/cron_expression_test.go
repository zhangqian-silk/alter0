package domain

import (
	"testing"
	"time"
)

func TestValidateCronExpression(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expectErr  bool
	}{
		{name: "every fifteen minutes", expression: "*/15 * * * *"},
		{name: "daily fixed time", expression: "30 9 * * *"},
		{name: "weekly fixed time", expression: "0 10 * * 1"},
		{name: "invalid field count", expression: "*/10 * * *", expectErr: true},
		{name: "invalid minute", expression: "70 * * * *", expectErr: true},
		{name: "invalid weekday", expression: "0 10 * * 8", expectErr: true},
	}

	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			err := ValidateCronExpression(item.expression)
			if item.expectErr && err == nil {
				t.Fatalf("expected error for expression %q", item.expression)
			}
			if !item.expectErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestNextCronFireAt(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	after := time.Date(2026, 3, 4, 1, 12, 0, 0, time.UTC)

	nextAt, err := NextCronFireAt("30 9 * * *", location, after)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := time.Date(2026, 3, 4, 1, 30, 0, 0, time.UTC)
	if !nextAt.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, nextAt)
	}
}
