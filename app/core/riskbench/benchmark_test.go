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
	payload, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal watchlist: %v", err)
	}
	if err := os.WriteFile(watchlistPath, payload, 0644); err != nil {
		t.Fatalf("write watchlist: %v", err)
	}
	if err := os.WriteFile(runbookPath, []byte("# runbook\n"), 0644); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	return watchlistPath, runbookPath
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
