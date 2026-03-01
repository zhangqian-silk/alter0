package riskbench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEvaluatePathsPassesWithFreshCoverage(t *testing.T) {
	now := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	watchlistPath, runbookPath := writeFixtureFiles(t, watchlistDocument{
		UpdatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		Items: []watchlistItem{
			{
				ID:           "provider-policy-drift",
				Category:     "provider_policy",
				Severity:     "high",
				Status:       "watch",
				NextReviewAt: now.Add(24 * time.Hour).Format(time.RFC3339),
				Owner:        "runtime",
			},
			{
				ID:           "plugin-provenance",
				Category:     "supply_chain",
				Severity:     "medium",
				Status:       "watch",
				NextReviewAt: now.Add(36 * time.Hour).Format(time.RFC3339),
				Owner:        "runtime",
			},
		},
	})

	report, err := EvaluatePaths(watchlistPath, runbookPath, Config{
		Now:           now,
		MaxStaleHours: 48,
		FailOnOverdue: true,
	})
	if err != nil {
		t.Fatalf("EvaluatePaths returned error: %v", err)
	}
	if !report.Gate.Passed {
		t.Fatalf("expected gate pass, got failures=%v warnings=%v", report.Gate.Failures, report.Gate.Warnings)
	}
	if report.Summary.OverdueItems != 0 {
		t.Fatalf("expected 0 overdue items, got %d", report.Summary.OverdueItems)
	}
	if report.Categories["provider_policy"].TotalItems != 1 {
		t.Fatalf("expected provider_policy coverage")
	}
	if report.Categories["supply_chain"].TotalItems != 1 {
		t.Fatalf("expected supply_chain coverage")
	}
}

func TestEvaluatePathsPassesWithScenarioAndCompetitorCoverage(t *testing.T) {
	now := time.Date(2026, 3, 2, 18, 0, 0, 0, time.UTC)
	watchlistPath, runbookPath := writeFixtureFiles(t, watchlistDocument{
		UpdatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		Items: []watchlistItem{
			{
				ID:           "provider-policy-drift",
				Category:     "provider_policy",
				Severity:     "high",
				Status:       "watch",
				NextReviewAt: now.Add(24 * time.Hour).Format(time.RFC3339),
				Owner:        "runtime",
			},
			{
				ID:           "plugin-provenance",
				Category:     "supply_chain",
				Severity:     "medium",
				Status:       "watch",
				NextReviewAt: now.Add(36 * time.Hour).Format(time.RFC3339),
				Owner:        "runtime",
			},
		},
	})

	dir := t.TempDir()
	scenarioPath := filepath.Join(dir, "scenario-benchmark-matrix.json")
	competitorPath := filepath.Join(dir, "competitor-tracking.json")

	writeJSONFixture(t, scenarioPath, scenarioMatrixDocument{
		UpdatedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
		Workloads: []scenarioWorkloadEntry{
			{
				ID:              "personal_assistant",
				Name:            "Personal Assistant",
				InputTemplate:   "daily briefing",
				LastBenchmarkAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
				Baseline:        scenarioResult{SuccessRate: 0.95, P95LatencyMS: 2800, AvgCostUSD: 0.014},
				Latest:          scenarioResult{SuccessRate: 0.94, P95LatencyMS: 2920, AvgCostUSD: 0.015},
			},
			{
				ID:              "team_collaboration",
				Name:            "Team Collaboration",
				InputTemplate:   "group mention follow-up",
				LastBenchmarkAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
				Baseline:        scenarioResult{SuccessRate: 0.93, P95LatencyMS: 3300, AvgCostUSD: 0.019},
				Latest:          scenarioResult{SuccessRate: 0.92, P95LatencyMS: 3410, AvgCostUSD: 0.02},
			},
			{
				ID:              "auto_patrol",
				Name:            "Auto Patrol",
				InputTemplate:   "heartbeat + release checks",
				LastBenchmarkAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
				Baseline:        scenarioResult{SuccessRate: 0.91, P95LatencyMS: 4100, AvgCostUSD: 0.024},
				Latest:          scenarioResult{SuccessRate: 0.9, P95LatencyMS: 4220, AvgCostUSD: 0.025},
			},
		},
	})

	writeJSONFixture(t, competitorPath, competitorTrackingDocument{
		UpdatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		Projects: []competitorTrackingRecord{
			{
				ID:            "openclaw",
				Name:          "OpenClaw",
				Repo:          "openclaw/openclaw",
				LastCheckedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
				Stars:         3000,
				Activity: competitorActivity{
					LastCommitAt: now.Add(-3 * time.Hour).Format(time.RFC3339),
					Commits30d:   90,
				},
				Release: competitorRelease{
					LastReleaseAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
					Releases90d:   4,
				},
				FeatureChanges: []competitorFeatureChange{{Date: "2026-03-01", Summary: "Browser relay enhancements", Area: "browser"}},
			},
			{
				ID:            "langchain",
				Name:          "LangChain",
				Repo:          "langchain-ai/langchain",
				LastCheckedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
				Stars:         100000,
				Activity: competitorActivity{
					LastCommitAt: now.Add(-4 * time.Hour).Format(time.RFC3339),
					Commits30d:   450,
				},
				Release: competitorRelease{
					LastReleaseAt: now.Add(-72 * time.Hour).Format(time.RFC3339),
					Releases90d:   8,
				},
				FeatureChanges: []competitorFeatureChange{{Date: "2026-02-28", Summary: "Agent observability update", Area: "agent-runtime"}},
			},
			{
				ID:            "dify",
				Name:          "Dify",
				Repo:          "langgenius/dify",
				LastCheckedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
				Stars:         80000,
				Activity: competitorActivity{
					LastCommitAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
					Commits30d:   170,
				},
				Release: competitorRelease{
					LastReleaseAt: now.Add(-48 * time.Hour).Format(time.RFC3339),
					Releases90d:   6,
				},
				FeatureChanges: []competitorFeatureChange{{Date: "2026-02-25", Summary: "Workflow governance refresh", Area: "workflow"}},
			},
		},
	})

	report, err := EvaluatePaths(watchlistPath, runbookPath, Config{
		Now:                    now,
		MaxStaleHours:          48,
		FailOnOverdue:          true,
		ScenarioMatrixPath:     scenarioPath,
		CompetitorTrackingPath: competitorPath,
		RequireScenarioMatrix:  true,
		RequireCompetitorTrack: true,
	})
	if err != nil {
		t.Fatalf("EvaluatePaths returned error: %v", err)
	}
	if !report.Gate.Passed {
		t.Fatalf("expected gate pass, got failures=%v warnings=%v", report.Gate.Failures, report.Gate.Warnings)
	}
	if report.ScenarioMatrix == nil || report.ScenarioMatrix.Summary.TotalWorkloads != 3 {
		t.Fatalf("expected 3 scenario workloads, got %#v", report.ScenarioMatrix)
	}
	if report.CompetitorTracking == nil || report.CompetitorTracking.Summary.TotalProjects != 3 {
		t.Fatalf("expected 3 competitor projects, got %#v", report.CompetitorTracking)
	}
}

func TestEvaluatePathsPassesWithCostThresholdHistoryCoverage(t *testing.T) {
	now := time.Date(2026, 3, 2, 18, 0, 0, 0, time.UTC)
	watchlistPath, runbookPath := writeFixtureFiles(t, watchlistDocument{
		UpdatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		Items: []watchlistItem{
			{
				ID:           "provider-policy-drift",
				Category:     "provider_policy",
				Severity:     "high",
				Status:       "watch",
				NextReviewAt: now.Add(24 * time.Hour).Format(time.RFC3339),
			},
			{
				ID:           "plugin-provenance",
				Category:     "supply_chain",
				Severity:     "medium",
				Status:       "watch",
				NextReviewAt: now.Add(24 * time.Hour).Format(time.RFC3339),
			},
		},
	})

	dir := t.TempDir()
	thresholdPath := filepath.Join(dir, "threshold-history-latest.json")
	writeJSONFixture(t, thresholdPath, costThresholdHistoryDocument{
		GeneratedAt: now.Add(-90 * time.Minute).Format(time.RFC3339),
		ThresholdGuidance: struct {
			Status         string `json:"status"`
			SampleSessions int    `json:"sample_sessions"`
			NeedsTuning    bool   `json:"needs_tuning"`
		}{
			Status:         "ok",
			SampleSessions: 3,
			NeedsTuning:    false,
		},
		Alerts: struct {
			HitRate struct {
				Samples                   int     `json:"samples"`
				SessionCostHotspot        float64 `json:"session_cost_hotspot"`
				SessionCompactionPressure float64 `json:"session_compaction_pressure"`
			} `json:"hit_rate"`
		}{
			HitRate: struct {
				Samples                   int     `json:"samples"`
				SessionCostHotspot        float64 `json:"session_cost_hotspot"`
				SessionCompactionPressure float64 `json:"session_compaction_pressure"`
			}{
				Samples:                   6,
				SessionCostHotspot:        0.33,
				SessionCompactionPressure: 0.17,
			},
		},
		Archive: struct {
			Week string `json:"week"`
		}{
			Week: "2026-W10",
		},
	})

	report, err := EvaluatePaths(watchlistPath, runbookPath, Config{
		Now:                     now,
		MaxStaleHours:           48,
		FailOnOverdue:           true,
		ThresholdHistoryPath:    thresholdPath,
		ThresholdMaxStaleDays:   8,
		RequireThresholdHistory: true,
	})
	if err != nil {
		t.Fatalf("EvaluatePaths returned error: %v", err)
	}
	if !report.Gate.Passed {
		t.Fatalf("expected gate pass, got failures=%v warnings=%v", report.Gate.Failures, report.Gate.Warnings)
	}
	if report.CostThresholdHistory == nil {
		t.Fatalf("expected cost threshold history in report")
	}
	if report.CostThresholdHistory.Summary.GuidanceStatus != "ok" {
		t.Fatalf("expected guidance status ok, got %#v", report.CostThresholdHistory.Summary.GuidanceStatus)
	}
}

func TestEvaluatePathsFailsWithStaleCostThresholdHistory(t *testing.T) {
	now := time.Date(2026, 3, 2, 18, 0, 0, 0, time.UTC)
	watchlistPath, runbookPath := writeFixtureFiles(t, watchlistDocument{
		UpdatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		Items: []watchlistItem{
			{
				ID:           "provider-policy-drift",
				Category:     "provider_policy",
				Severity:     "high",
				Status:       "watch",
				NextReviewAt: now.Add(24 * time.Hour).Format(time.RFC3339),
			},
			{
				ID:           "plugin-provenance",
				Category:     "supply_chain",
				Severity:     "medium",
				Status:       "watch",
				NextReviewAt: now.Add(24 * time.Hour).Format(time.RFC3339),
			},
		},
	})

	dir := t.TempDir()
	thresholdPath := filepath.Join(dir, "threshold-history-latest.json")
	writeJSONFixture(t, thresholdPath, costThresholdHistoryDocument{
		GeneratedAt: now.Add(-12 * 24 * time.Hour).Format(time.RFC3339),
		ThresholdGuidance: struct {
			Status         string `json:"status"`
			SampleSessions int    `json:"sample_sessions"`
			NeedsTuning    bool   `json:"needs_tuning"`
		}{
			Status:         "ok",
			SampleSessions: 2,
			NeedsTuning:    false,
		},
		Alerts: struct {
			HitRate struct {
				Samples                   int     `json:"samples"`
				SessionCostHotspot        float64 `json:"session_cost_hotspot"`
				SessionCompactionPressure float64 `json:"session_compaction_pressure"`
			} `json:"hit_rate"`
		}{
			HitRate: struct {
				Samples                   int     `json:"samples"`
				SessionCostHotspot        float64 `json:"session_cost_hotspot"`
				SessionCompactionPressure float64 `json:"session_compaction_pressure"`
			}{
				Samples:                   4,
				SessionCostHotspot:        0.5,
				SessionCompactionPressure: 0.25,
			},
		},
		Archive: struct {
			Week string `json:"week"`
		}{
			Week: "2026-W08",
		},
	})

	report, err := EvaluatePaths(watchlistPath, runbookPath, Config{
		Now:                     now,
		MaxStaleHours:           48,
		FailOnOverdue:           true,
		ThresholdHistoryPath:    thresholdPath,
		ThresholdMaxStaleDays:   8,
		RequireThresholdHistory: true,
	})
	if err != nil {
		t.Fatalf("EvaluatePaths returned error: %v", err)
	}
	if report.Gate.Passed {
		t.Fatalf("expected gate fail")
	}
	if !containsSubstring(report.Gate.Failures, "cost threshold history: cost threshold history is stale") {
		t.Fatalf("expected stale threshold history failure, got %v", report.Gate.Failures)
	}
}

func TestEvaluatePathsFailsOnMissingCategoryAndOverdue(t *testing.T) {
	now := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	watchlistPath, runbookPath := writeFixtureFiles(t, watchlistDocument{
		UpdatedAt: now.Add(-4 * time.Hour).Format(time.RFC3339),
		Items: []watchlistItem{
			{
				ID:           "provider-policy-drift",
				Category:     "provider_policy",
				Severity:     "high",
				Status:       "watch",
				NextReviewAt: now.Add(-6 * time.Hour).Format(time.RFC3339),
				Owner:        "runtime",
			},
		},
	})

	report, err := EvaluatePaths(watchlistPath, runbookPath, Config{
		Now:           now,
		MaxStaleHours: 48,
		FailOnOverdue: true,
	})
	if err != nil {
		t.Fatalf("EvaluatePaths returned error: %v", err)
	}
	if report.Gate.Passed {
		t.Fatalf("expected gate fail")
	}
	if !contains(report.Gate.Failures, "missing required risk categories: supply_chain") {
		t.Fatalf("expected missing category failure, got %v", report.Gate.Failures)
	}
	if !contains(report.Gate.Failures, "overdue risk items detected: 1") {
		t.Fatalf("expected overdue failure, got %v", report.Gate.Failures)
	}
	if len(report.Drifts) != 1 {
		t.Fatalf("expected one drift event, got %d", len(report.Drifts))
	}
	if report.Drifts[0].DriftLevel != "critical" {
		t.Fatalf("expected critical drift level, got %s", report.Drifts[0].DriftLevel)
	}
}

func TestEvaluatePathsFailsScenarioAndCompetitorGates(t *testing.T) {
	now := time.Date(2026, 3, 2, 18, 0, 0, 0, time.UTC)
	watchlistPath, runbookPath := writeFixtureFiles(t, watchlistDocument{
		UpdatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		Items: []watchlistItem{
			{
				ID:           "provider-policy-drift",
				Category:     "provider_policy",
				Severity:     "high",
				Status:       "watch",
				NextReviewAt: now.Add(24 * time.Hour).Format(time.RFC3339),
			},
			{
				ID:           "plugin-provenance",
				Category:     "supply_chain",
				Severity:     "medium",
				Status:       "watch",
				NextReviewAt: now.Add(24 * time.Hour).Format(time.RFC3339),
			},
		},
	})

	dir := t.TempDir()
	scenarioPath := filepath.Join(dir, "scenario-benchmark-matrix.json")
	competitorPath := filepath.Join(dir, "competitor-tracking.json")

	writeJSONFixture(t, scenarioPath, scenarioMatrixDocument{
		UpdatedAt: now.Add(-60 * 24 * time.Hour).Format(time.RFC3339),
		Workloads: []scenarioWorkloadEntry{
			{
				ID:              "personal_assistant",
				Name:            "Personal Assistant",
				InputTemplate:   "daily briefing",
				LastBenchmarkAt: now.Add(-60 * 24 * time.Hour).Format(time.RFC3339),
				Baseline:        scenarioResult{SuccessRate: 0.95, P95LatencyMS: 2800, AvgCostUSD: 0.014},
				Latest:          scenarioResult{SuccessRate: 0.9, P95LatencyMS: 3200, AvgCostUSD: 0.017},
			},
			{
				ID:              "team_collaboration",
				Name:            "Team Collaboration",
				InputTemplate:   "group mention follow-up",
				LastBenchmarkAt: now.Add(-60 * 24 * time.Hour).Format(time.RFC3339),
				Baseline:        scenarioResult{SuccessRate: 0.93, P95LatencyMS: 3300, AvgCostUSD: 0.019},
				Latest:          scenarioResult{SuccessRate: 0.89, P95LatencyMS: 3800, AvgCostUSD: 0.023},
			},
		},
	})

	writeJSONFixture(t, competitorPath, competitorTrackingDocument{
		UpdatedAt: now.Add(-40 * 24 * time.Hour).Format(time.RFC3339),
		Projects: []competitorTrackingRecord{
			{
				ID:            "openclaw",
				Name:          "OpenClaw",
				Repo:          "openclaw/openclaw",
				LastCheckedAt: now.Add(-40 * 24 * time.Hour).Format(time.RFC3339),
				Stars:         3000,
				Activity: competitorActivity{
					LastCommitAt: now.Add(-7 * 24 * time.Hour).Format(time.RFC3339),
					Commits30d:   90,
				},
				Release: competitorRelease{
					LastReleaseAt: now.Add(-20 * 24 * time.Hour).Format(time.RFC3339),
					Releases90d:   4,
				},
				FeatureChanges: []competitorFeatureChange{},
			},
			{
				ID:            "langchain",
				Name:          "LangChain",
				Repo:          "langchain-ai/langchain",
				LastCheckedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
				Stars:         100000,
				Activity: competitorActivity{
					LastCommitAt: now.Add(-6 * time.Hour).Format(time.RFC3339),
					Commits30d:   450,
				},
				Release: competitorRelease{
					LastReleaseAt: now.Add(-48 * time.Hour).Format(time.RFC3339),
					Releases90d:   8,
				},
				FeatureChanges: []competitorFeatureChange{{Date: "2026-03-01", Summary: "Agent observability update", Area: "agent-runtime"}},
			},
		},
	})

	report, err := EvaluatePaths(watchlistPath, runbookPath, Config{
		Now:                    now,
		MaxStaleHours:          48,
		FailOnOverdue:          true,
		ScenarioMatrixPath:     scenarioPath,
		CompetitorTrackingPath: competitorPath,
		ScenarioMaxStaleDays:   45,
		CompetitorMaxStaleDays: 31,
		RequireScenarioMatrix:  true,
		RequireCompetitorTrack: true,
	})
	if err != nil {
		t.Fatalf("EvaluatePaths returned error: %v", err)
	}
	if report.Gate.Passed {
		t.Fatalf("expected gate fail")
	}
	if !containsSubstring(report.Gate.Failures, "scenario matrix: scenario benchmark matrix is stale") {
		t.Fatalf("expected scenario stale failure, got %v", report.Gate.Failures)
	}
	if !containsSubstring(report.Gate.Failures, "scenario matrix: missing required workloads: auto_patrol") {
		t.Fatalf("expected missing workload failure, got %v", report.Gate.Failures)
	}
	if !containsSubstring(report.Gate.Failures, "competitor tracking: competitor tracking requires at least 3 projects") {
		t.Fatalf("expected competitor project-count failure, got %v", report.Gate.Failures)
	}
	if !containsSubstring(report.Gate.Failures, "competitor tracking: competitor tracking has stale project entries") {
		t.Fatalf("expected competitor stale failure, got %v", report.Gate.Failures)
	}
	if !containsSubstring(report.Gate.Failures, "competitor tracking: competitor tracking has projects without feature_changes") {
		t.Fatalf("expected competitor feature-change failure, got %v", report.Gate.Failures)
	}
}

func TestEvaluatePathsFailsWhenRunbookMissing(t *testing.T) {
	now := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	watchlistPath, _ := writeFixtureFiles(t, watchlistDocument{
		UpdatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		Items: []watchlistItem{
			{
				ID:           "provider-policy-drift",
				Category:     "provider_policy",
				Severity:     "medium",
				Status:       "watch",
				NextReviewAt: now.Add(12 * time.Hour).Format(time.RFC3339),
			},
			{
				ID:           "plugin-provenance",
				Category:     "supply_chain",
				Severity:     "medium",
				Status:       "watch",
				NextReviewAt: now.Add(16 * time.Hour).Format(time.RFC3339),
			},
		},
	})

	report, err := EvaluatePaths(watchlistPath, filepath.Join(t.TempDir(), "missing-runbook.md"), Config{
		Now:           now,
		MaxStaleHours: 48,
		FailOnOverdue: true,
	})
	if err != nil {
		t.Fatalf("EvaluatePaths returned error: %v", err)
	}
	if report.Gate.Passed {
		t.Fatalf("expected gate fail when runbook is missing")
	}
	if len(report.Gate.Failures) == 0 || !strings.Contains(report.Gate.Failures[0], "drift triage runbook is unavailable") {
		t.Fatalf("expected runbook failure, got %v", report.Gate.Failures)
	}
}

func writeFixtureFiles(t *testing.T, doc watchlistDocument) (string, string) {
	t.Helper()
	dir := t.TempDir()
	watchlistPath := filepath.Join(dir, "risk-watchlist.json")
	runbookPath := filepath.Join(dir, "risk-runbook.md")
	writeJSONFixture(t, watchlistPath, doc)
	if err := os.WriteFile(runbookPath, []byte("# runbook\n"), 0644); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	return watchlistPath, runbookPath
}

func writeJSONFixture(t *testing.T, path string, payload any) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsSubstring(items []string, target string) bool {
	for _, item := range items {
		if strings.Contains(item, target) {
			return true
		}
	}
	return false
}
