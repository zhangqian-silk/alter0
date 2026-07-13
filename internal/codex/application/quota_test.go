package application_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	codexapp "alter0/internal/codex/application"
)

func TestQueryQuotaClassifiesWindowsByDuration(t *testing.T) {
	now := time.Date(2026, 7, 13, 2, 0, 0, 0, time.UTC)
	rawAuth := buildServiceOAuthAuth(t, serviceOAuthInput{
		Name:      "Runtime Account",
		Email:     "runtime@example.com",
		UserID:    "user-runtime",
		AccountID: "acct-runtime",
		Plan:      "pro",
		ExpiresAt: now.Add(24 * time.Hour),
	})

	tests := []struct {
		name       string
		body       string
		wantHourly int
		wantWeekly int
		hasHourly  bool
	}{
		{
			name: "classifies reversed five hour and weekly windows",
			body: `{
				"plan_type":"pro",
				"rate_limit":{
					"primary_window":{"used_percent":7,"limit_window_seconds":604800,"reset_at":1784487499},
					"secondary_window":{"used_percent":11,"limit_window_seconds":18000,"reset_at":1783908000}
				}
			}`,
			wantHourly: 89,
			wantWeekly: 93,
			hasHourly:  true,
		},
		{
			name: "omits temporarily absent five hour window",
			body: `{
				"plan_type":"pro",
				"rate_limit":{
					"primary_window":{"used_percent":7,"limit_window_seconds":604800,"reset_at":1784487499},
					"secondary_window":null
				}
			}`,
			wantWeekly: 93,
			hasHourly:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: quotaRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			})}

			status, _, err := codexapp.QueryQuota(rawAuth, codexapp.QuotaQueryOptions{Now: now, Client: client})
			if err != nil {
				t.Fatalf("QueryQuota returned error: %v", err)
			}
			hasHourly := status.Hourly != nil
			if hasHourly != tt.hasHourly {
				t.Fatalf("hourly presence = %v, want %v", hasHourly, tt.hasHourly)
			}
			if hasHourly && status.Hourly.RemainingPercent != tt.wantHourly {
				t.Fatalf("hourly remaining = %d, want %d", status.Hourly.RemainingPercent, tt.wantHourly)
			}
			if status.Weekly == nil {
				t.Fatal("weekly quota is missing")
			}
			if status.Weekly.RemainingPercent != tt.wantWeekly {
				t.Fatalf("weekly remaining = %d, want %d", status.Weekly.RemainingPercent, tt.wantWeekly)
			}
		})
	}
}

type quotaRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn quotaRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
