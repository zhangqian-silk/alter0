package channelchaos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildCalibrationReportComputesAdoptionAndReduction(t *testing.T) {
	now := time.Date(2026, time.March, 2, 1, 0, 0, 0, time.UTC)
	root := t.TempDir()

	candidateRoot := filepath.Join(root, "candidates")
	writeJSON(t, filepath.Join(candidateRoot, "2026-W09", "candidates-a.json"), CandidateReport{
		GeneratedAt: now.Add(-72 * time.Hour).Format(time.RFC3339),
		Candidates: []Scenario{
			{ID: "a", SourceCandidate: "trace:slack:critical:true"},
			{ID: "b", SourceCandidate: "trace:http:critical:false"},
		},
	})
	writeJSON(t, filepath.Join(candidateRoot, "2026-W09", "candidates-b.json"), CandidateReport{
		GeneratedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
		Candidates: []Scenario{
			{ID: "c", SourceCandidate: "trace:http:critical:false"},
			{ID: "d", SourceCandidate: "trace:telegram:degraded:false"},
		},
	})

	matrixPath := filepath.Join(root, "channel-chaos-matrix.json")
	writeJSON(t, matrixPath, Matrix{
		UpdatedAt: now.Format(time.RFC3339),
		Scenarios: []Scenario{
			{
				ID:              "scenario-1",
				SourceCandidate: "trace:http:critical:false",
				Events:          []TraceEvent{{ChannelID: "http", Event: "agent_process", Status: "error"}},
				Expect:          Expectation{Status: "critical"},
			},
		},
	})

	historyRoot := filepath.Join(root, "threshold-history")
	writeJSON(t, filepath.Join(historyRoot, "2026-W09", "history-a.json"), map[string]interface{}{
		"generated_at": now.Add(-96 * time.Hour).Format(time.RFC3339),
		"alerts": map[string]interface{}{
			"hit_rate": map[string]float64{
				"session_cost_hotspot":        0.8,
				"session_compaction_pressure": 0.4,
			},
		},
	})
	writeJSON(t, filepath.Join(historyRoot, "2026-W09", "history-b.json"), map[string]interface{}{
		"generated_at": now.Add(-6 * time.Hour).Format(time.RFC3339),
		"alerts": map[string]interface{}{
			"hit_rate": map[string]float64{
				"session_cost_hotspot":        0.2,
				"session_compaction_pressure": 0.2,
			},
		},
	})

	reconcileRoot := filepath.Join(root, "threshold-reconcile")
	writeJSON(t, filepath.Join(reconcileRoot, "2026-W09", "reconcile-a.json"), map[string]interface{}{
		"generated_at": now.Add(-48 * time.Hour).Format(time.RFC3339),
		"applied":      false,
		"plan": map[string]string{
			"status": "ready",
		},
	})
	writeJSON(t, filepath.Join(reconcileRoot, "2026-W09", "reconcile-b.json"), map[string]interface{}{
		"generated_at": now.Add(-12 * time.Hour).Format(time.RFC3339),
		"applied":      true,
		"plan": map[string]string{
			"status": "applied",
		},
	})

	report, err := BuildCalibrationReport(CalibrationOptions{
		Now:                    now,
		Window:                 7 * 24 * time.Hour,
		CandidateArchiveRoot:   candidateRoot,
		MatrixPath:             matrixPath,
		ThresholdHistoryRoot:   historyRoot,
		ThresholdReconcileRoot: reconcileRoot,
	})
	if err != nil {
		t.Fatalf("BuildCalibrationReport returned error: %v", err)
	}

	if report.Candidate.Reports != 2 {
		t.Fatalf("expected candidate reports=2, got %d", report.Candidate.Reports)
	}
	if report.Candidate.UniqueCandidates != 3 {
		t.Fatalf("expected unique candidates=3, got %d", report.Candidate.UniqueCandidates)
	}
	if report.Candidate.AdoptedCandidates != 1 {
		t.Fatalf("expected adopted candidates=1, got %d", report.Candidate.AdoptedCandidates)
	}
	if report.Candidate.AdoptionRate != 0.333 {
		t.Fatalf("expected adoption rate 0.333, got %f", report.Candidate.AdoptionRate)
	}
	if len(report.Candidate.PendingCandidates) != 2 {
		t.Fatalf("expected 2 pending candidates, got %#v", report.Candidate.PendingCandidates)
	}

	if report.FalsePositive.Status != "improved" {
		t.Fatalf("expected false-positive status improved, got %q", report.FalsePositive.Status)
	}
	if report.FalsePositive.ReductionRate != 0.667 {
		t.Fatalf("expected reduction rate 0.667, got %f", report.FalsePositive.ReductionRate)
	}

	if report.Threshold.Samples != 2 {
		t.Fatalf("expected threshold samples=2, got %d", report.Threshold.Samples)
	}
	if report.Threshold.ReadyRate != 1 {
		t.Fatalf("expected ready rate 1, got %f", report.Threshold.ReadyRate)
	}
	if report.Threshold.AppliedRate != 0.5 {
		t.Fatalf("expected applied rate 0.5, got %f", report.Threshold.AppliedRate)
	}
}

func TestBuildCalibrationReportNoData(t *testing.T) {
	now := time.Date(2026, time.March, 2, 1, 0, 0, 0, time.UTC)
	root := t.TempDir()
	matrixPath := filepath.Join(root, "channel-chaos-matrix.json")
	writeJSON(t, matrixPath, Matrix{
		UpdatedAt: now.Format(time.RFC3339),
		Scenarios: []Scenario{{
			ID:     "manual",
			Events: []TraceEvent{{ChannelID: "cli", Event: "inbound_received", Status: "ok"}},
			Expect: Expectation{Status: "monitoring"},
		}},
	})

	report, err := BuildCalibrationReport(CalibrationOptions{
		Now:                    now,
		Window:                 24 * time.Hour,
		CandidateArchiveRoot:   filepath.Join(root, "missing-candidates"),
		MatrixPath:             matrixPath,
		ThresholdHistoryRoot:   filepath.Join(root, "missing-history"),
		ThresholdReconcileRoot: filepath.Join(root, "missing-reconcile"),
	})
	if err != nil {
		t.Fatalf("BuildCalibrationReport returned error: %v", err)
	}
	if report.Candidate.UniqueCandidates != 0 {
		t.Fatalf("expected no candidates, got %d", report.Candidate.UniqueCandidates)
	}
	if report.FalsePositive.Status != "no_data" {
		t.Fatalf("expected false-positive no_data, got %q", report.FalsePositive.Status)
	}
	if report.Threshold.Samples != 0 {
		t.Fatalf("expected no threshold samples, got %d", report.Threshold.Samples)
	}
	if len(report.Notes) < 3 {
		t.Fatalf("expected explanatory notes, got %#v", report.Notes)
	}
}

func writeJSON(t *testing.T, path string, value interface{}) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0644); err != nil {
		t.Fatalf("write json failed: %v", err)
	}
}
