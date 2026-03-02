package channelchaos

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"alter0/app/core/runtime"
)

func TestLoadMatrixRequiresScenarios(t *testing.T) {
	path := filepath.Join(t.TempDir(), "matrix.json")
	if err := os.WriteFile(path, []byte(`{"updated_at":"2026-03-02T00:00:00Z","scenarios":[]}`), 0644); err != nil {
		t.Fatalf("write matrix failed: %v", err)
	}

	_, err := LoadMatrix(path)
	if err == nil {
		t.Fatal("expected error when scenarios are empty")
	}
}

func TestRunMatrixPassesScenario(t *testing.T) {
	one := 1
	matrix := Matrix{
		UpdatedAt: "2026-03-02T00:00:00Z",
		Scenarios: []Scenario{
			{
				ID:          "single-channel-disconnect-with-fallback",
				Description: "single disconnect keeps healthy fallback",
				Events: []TraceEvent{
					{OffsetSeconds: -30, ChannelID: "slack", Event: "channel_disconnected", Status: "error", Detail: "socket closed"},
					{OffsetSeconds: -20, ChannelID: "telegram", Event: "inbound_received", Status: "ok"},
				},
				Expect: Expectation{
					Status:                "critical",
					MinDegradedChannels:   &one,
					MinFallbackCandidates: &one,
					AlertCodes:            []string{"channel_disconnected", "channel_degradation"},
				},
			},
		},
	}

	report := Run(context.Background(), matrix)
	if !report.Passed {
		payload, _ := json.Marshal(report)
		t.Fatalf("expected report pass, got %s", payload)
	}
	if report.FailedCount != 0 {
		t.Fatalf("expected failed_count=0, got %d", report.FailedCount)
	}
}

func TestRunMatrixSupportsThresholdPolicySuppression(t *testing.T) {
	one := 1
	zero := 0
	matrix := Matrix{
		Scenarios: []Scenario{
			{
				ID: "channel-override-suppresses-noise",
				Events: []TraceEvent{
					{OffsetSeconds: -30, ChannelID: "slack", Event: "agent_process", Status: "error", Detail: "retryable"},
					{OffsetSeconds: -20, ChannelID: "slack", Event: "agent_process", Status: "ok"},
					{OffsetSeconds: -10, ChannelID: "cli", Event: "inbound_received", Status: "ok"},
				},
				Thresholds: ThresholdPolicy{
					Default: runtime.ChannelDegradationThresholds{
						MinEvents:                     1,
						WarningErrorRateThreshold:     0.001,
						CriticalErrorRateThreshold:    0.5,
						CriticalErrorCountThreshold:   3,
						CriticalDisconnectedThreshold: 1,
					},
					Overrides: map[string]runtime.ChannelDegradationThresholds{
						"slack": {
							MinEvents:                     5,
							WarningErrorRateThreshold:     0.5,
							CriticalErrorRateThreshold:    0.8,
							CriticalErrorCountThreshold:   4,
							CriticalDisconnectedThreshold: 1,
						},
					},
				},
				Expect: Expectation{
					Status:                  "monitoring",
					MinSuppressedChannels:   &one,
					MaxDegradedChannels:     &zero,
					MinFallbackCandidates:   &one,
					RequiredThresholdPolicy: []string{"channel:slack"},
				},
			},
		},
	}

	report := Run(context.Background(), matrix)
	if !report.Passed {
		payload, _ := json.Marshal(report)
		t.Fatalf("expected report pass, got %s", payload)
	}
}

func TestRunMatrixFailsWhenFallbackMissing(t *testing.T) {
	one := 1
	matrix := Matrix{
		Scenarios: []Scenario{
			{
				ID: "all-channels-degraded",
				Events: []TraceEvent{
					{OffsetSeconds: -30, ChannelID: "slack", Event: "channel_disconnected", Status: "error", Detail: "socket closed"},
					{OffsetSeconds: -20, ChannelID: "telegram", Event: "agent_process", Status: "error", Detail: "rate limited"},
				},
				Expect: Expectation{
					MinFallbackCandidates: &one,
				},
			},
		},
	}

	report := Run(context.Background(), matrix)
	if report.Passed {
		t.Fatal("expected report to fail when fallback is missing")
	}
	if report.FailedCount != 1 {
		t.Fatalf("expected one failed scenario, got %d", report.FailedCount)
	}
	if report.Results[0].Failure == "" {
		t.Fatalf("expected failure reason, got %+v", report.Results[0])
	}
}
