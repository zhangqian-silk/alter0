package riskbench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CostThresholdHistoryReport struct {
	Path        string                      `json:"path"`
	GeneratedAt string                      `json:"generated_at,omitempty"`
	AgeDays     int                         `json:"age_days,omitempty"`
	Summary     CostThresholdHistorySummary `json:"summary"`
	Gate        GateResult                  `json:"gate"`
}

type CostThresholdHistorySummary struct {
	Status                           string  `json:"status"`
	GuidanceStatus                   string  `json:"guidance_status,omitempty"`
	SampleSessions                   int     `json:"sample_sessions"`
	NeedsTuning                      bool    `json:"needs_tuning"`
	AlertSamples                     int     `json:"alert_samples"`
	SessionCostHotspotHitRate        float64 `json:"session_cost_hotspot_hit_rate"`
	SessionCompactionPressureHitRate float64 `json:"session_compaction_pressure_hit_rate"`
	ArchiveWeek                      string  `json:"archive_week,omitempty"`
}

type costThresholdHistoryDocument struct {
	GeneratedAt       string `json:"generated_at"`
	ThresholdGuidance struct {
		Status         string `json:"status"`
		SampleSessions int    `json:"sample_sessions"`
		NeedsTuning    bool   `json:"needs_tuning"`
	} `json:"threshold_guidance"`
	Alerts struct {
		HitRate struct {
			Samples                   int     `json:"samples"`
			SessionCostHotspot        float64 `json:"session_cost_hotspot"`
			SessionCompactionPressure float64 `json:"session_compaction_pressure"`
		} `json:"hit_rate"`
	} `json:"alerts"`
	Archive struct {
		Week string `json:"week"`
	} `json:"archive"`
}

func evaluateCostThresholdHistory(cfg Config, now time.Time) (*CostThresholdHistoryReport, []string, []string) {
	trimmedPath := strings.TrimSpace(cfg.ThresholdHistoryPath)
	if trimmedPath == "" {
		if cfg.RequireThresholdHistory {
			return nil, []string{"cost threshold history path is empty"}, nil
		}
		return nil, nil, nil
	}
	resolvedPath := filepath.Clean(trimmedPath)
	payload, err := os.ReadFile(resolvedPath)
	if err != nil {
		message := fmt.Sprintf("cost threshold history snapshot is unavailable: %s", resolvedPath)
		if cfg.RequireThresholdHistory {
			return nil, []string{message}, nil
		}
		return nil, nil, []string{message}
	}

	var doc costThresholdHistoryDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		message := fmt.Sprintf("cost threshold history snapshot JSON is invalid: %s", resolvedPath)
		if cfg.RequireThresholdHistory {
			return nil, []string{message}, nil
		}
		return nil, nil, []string{message}
	}

	report := &CostThresholdHistoryReport{
		Path: resolvedPath,
		Gate: GateResult{
			Failures: make([]string, 0),
			Warnings: make([]string, 0),
		},
	}

	report.Summary.GuidanceStatus = strings.TrimSpace(doc.ThresholdGuidance.Status)
	report.Summary.SampleSessions = doc.ThresholdGuidance.SampleSessions
	report.Summary.NeedsTuning = doc.ThresholdGuidance.NeedsTuning
	report.Summary.AlertSamples = doc.Alerts.HitRate.Samples
	report.Summary.SessionCostHotspotHitRate = doc.Alerts.HitRate.SessionCostHotspot
	report.Summary.SessionCompactionPressureHitRate = doc.Alerts.HitRate.SessionCompactionPressure
	report.Summary.ArchiveWeek = strings.TrimSpace(doc.Archive.Week)

	if generatedAt, ok := parseRFC3339(doc.GeneratedAt); ok {
		report.GeneratedAt = generatedAt.Format(time.RFC3339)
		report.AgeDays = int(now.Sub(generatedAt).Hours() / 24)
		if report.AgeDays > cfg.ThresholdMaxStaleDays {
			report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("cost threshold history is stale: age=%dd max=%dd", report.AgeDays, cfg.ThresholdMaxStaleDays))
		}
	} else {
		report.Gate.Failures = append(report.Gate.Failures, "cost threshold history generated_at is missing or invalid")
	}

	if report.Summary.GuidanceStatus == "" {
		report.Summary.GuidanceStatus = "missing"
	}
	if report.Summary.GuidanceStatus != "ok" {
		report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("threshold guidance status is %s", report.Summary.GuidanceStatus))
	}
	if report.Summary.SampleSessions == 0 {
		report.Gate.Warnings = append(report.Gate.Warnings, "threshold guidance has zero heavy-session samples")
	}
	if report.Summary.NeedsTuning {
		report.Gate.Warnings = append(report.Gate.Warnings, "threshold guidance indicates tuning is required")
	}
	if report.Summary.AlertSamples > 0 {
		if report.Summary.SessionCostHotspotHitRate >= 0.6 {
			report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("session_cost_hotspot alert hit-rate is high: %.3f", report.Summary.SessionCostHotspotHitRate))
		}
		if report.Summary.SessionCompactionPressureHitRate >= 0.6 {
			report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("session_compaction_pressure alert hit-rate is high: %.3f", report.Summary.SessionCompactionPressureHitRate))
		}
	}

	report.Gate.Passed = len(report.Gate.Failures) == 0
	if report.Gate.Passed {
		report.Summary.Status = "pass"
	} else {
		report.Summary.Status = "fail"
	}

	return report, prefixMessages("cost threshold history: ", report.Gate.Failures), prefixMessages("cost threshold history: ", report.Gate.Warnings)
}
