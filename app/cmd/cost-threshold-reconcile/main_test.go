package main

import "testing"

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
