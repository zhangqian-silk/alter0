package main

import (
	"testing"
	"time"
)

func TestBuildPlanSkipsWhenGuidanceNotReady(t *testing.T) {
	current := thresholdValues{
		SessionCostShareAlertThreshold:  0.35,
		PromptOutputRatioAlertThreshold: 6.0,
		HeavySessionMinTokens:           1200,
	}
	guidance := thresholdGuidance{Status: "insufficient_data"}

	plan := buildPlan(current, guidance, planOptions{})

	if plan.Status != "skipped" {
		t.Fatalf("expected skipped status, got %q", plan.Status)
	}
	if plan.Proposed != current {
		t.Fatalf("expected unchanged proposal, got %#v", plan.Proposed)
	}
}

func TestBuildPlanCapsDeltaAndKeepsStableStep(t *testing.T) {
	current := thresholdValues{
		SessionCostShareAlertThreshold:  0.35,
		PromptOutputRatioAlertThreshold: 6.0,
		HeavySessionMinTokens:           1200,
	}
	guidance := thresholdGuidance{
		Status:      "ok",
		NeedsTuning: true,
		Recommended: struct {
			SessionCostShare  float64 `json:"session_cost_share"`
			PromptOutputRatio float64 `json:"prompt_output_ratio"`
		}{
			SessionCostShare:  0.85,
			PromptOutputRatio: 11.0,
		},
		RequiredMinTokens: 1600,
	}

	plan := buildPlan(current, guidance, planOptions{MaxShareStep: 0.05, MaxRatioStep: 1.5})

	if plan.Status != "ready" {
		t.Fatalf("expected ready status, got %q (%s)", plan.Status, plan.Reason)
	}
	if !plan.CappedDelta {
		t.Fatalf("expected capped delta=true")
	}
	if plan.Proposed.SessionCostShareAlertThreshold != 0.4 {
		t.Fatalf("expected share threshold 0.4, got %f", plan.Proposed.SessionCostShareAlertThreshold)
	}
	if plan.Proposed.PromptOutputRatioAlertThreshold != 7.5 {
		t.Fatalf("expected prompt ratio threshold 7.5, got %f", plan.Proposed.PromptOutputRatioAlertThreshold)
	}
	if plan.Proposed.HeavySessionMinTokens != 1600 {
		t.Fatalf("expected min tokens 1600, got %d", plan.Proposed.HeavySessionMinTokens)
	}
}

func TestBuildPlanNoChangeWhenAlreadyAligned(t *testing.T) {
	current := thresholdValues{
		SessionCostShareAlertThreshold:  0.42,
		PromptOutputRatioAlertThreshold: 7.5,
		HeavySessionMinTokens:           1200,
	}
	guidance := thresholdGuidance{
		Status:      "ok",
		NeedsTuning: false,
		Recommended: struct {
			SessionCostShare  float64 `json:"session_cost_share"`
			PromptOutputRatio float64 `json:"prompt_output_ratio"`
		}{
			SessionCostShare:  0.42,
			PromptOutputRatio: 7.5,
		},
		RequiredMinTokens: 1200,
	}

	plan := buildPlan(current, guidance, planOptions{})

	if plan.Status != "no_change" {
		t.Fatalf("expected no_change status, got %q", plan.Status)
	}
}

func TestSummarizeCadenceUsesRecentWindowAndReadyStreak(t *testing.T) {
	history := []reconcileSample{
		{Timestamp: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), Status: "skipped", Applied: false},
		{Timestamp: time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC), Status: "ready", Applied: false},
		{Timestamp: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), Status: "ready", Applied: false},
	}
	current := reconcileSample{Timestamp: time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC), Status: "applied", Applied: true}

	summary := summarizeCadence(history, current, 3)

	if summary.Samples != 3 {
		t.Fatalf("expected samples=3, got %d", summary.Samples)
	}
	if summary.StatusCount["ready"] != 2 || summary.StatusCount["applied"] != 1 {
		t.Fatalf("unexpected status counts: %#v", summary.StatusCount)
	}
	if summary.ReadyRate != 1 {
		t.Fatalf("expected ready rate 1, got %f", summary.ReadyRate)
	}
	if summary.AppliedRate != 0.333 {
		t.Fatalf("expected applied rate 0.333, got %f", summary.AppliedRate)
	}
	if summary.ReadyStreak != 3 {
		t.Fatalf("expected ready streak 3, got %d", summary.ReadyStreak)
	}
}

func TestSummarizeCadenceEmpty(t *testing.T) {
	summary := summarizeCadence(nil, reconcileSample{}, 10)
	if summary.Samples != 0 {
		t.Fatalf("expected samples=0, got %d", summary.Samples)
	}
	if len(summary.StatusCount) != 0 {
		t.Fatalf("expected empty status count, got %#v", summary.StatusCount)
	}
}
