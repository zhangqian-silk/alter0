package riskbench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ScenarioMatrixReport struct {
	Path      string                            `json:"path"`
	UpdatedAt string                            `json:"updated_at,omitempty"`
	AgeDays   int                               `json:"age_days,omitempty"`
	Summary   ScenarioMatrixSummary             `json:"summary"`
	Workloads map[string]ScenarioWorkloadResult `json:"workloads"`
	Gate      GateResult                        `json:"gate"`
}

type ScenarioMatrixSummary struct {
	Status           string   `json:"status"`
	TotalWorkloads   int      `json:"total_workloads"`
	MissingWorkloads []string `json:"missing_workloads,omitempty"`
	StaleWorkloads   int      `json:"stale_workloads"`
}

type ScenarioWorkloadResult struct {
	Name            string  `json:"name"`
	InputTemplate   string  `json:"input_template,omitempty"`
	LastBenchmarkAt string  `json:"last_benchmark_at,omitempty"`
	SuccessRate     float64 `json:"success_rate"`
	P95LatencyMS    int     `json:"p95_latency_ms"`
	AvgCostUSD      float64 `json:"avg_cost_usd"`
	DriftLevel      string  `json:"drift_level"`
}

type CompetitorTrackingReport struct {
	Path      string                              `json:"path"`
	UpdatedAt string                              `json:"updated_at,omitempty"`
	AgeDays   int                                 `json:"age_days,omitempty"`
	Summary   CompetitorTrackingSummary           `json:"summary"`
	Projects  map[string]CompetitorProjectSummary `json:"projects"`
	Gate      GateResult                          `json:"gate"`
}

type CompetitorTrackingSummary struct {
	Status               string `json:"status"`
	TotalProjects        int    `json:"total_projects"`
	ActiveProjects       int    `json:"active_projects"`
	StaleProjects        int    `json:"stale_projects"`
	MissingChangeEntries int    `json:"missing_change_entries"`
}

type CompetitorProjectSummary struct {
	Name           string `json:"name"`
	Repo           string `json:"repo"`
	LastCheckedAt  string `json:"last_checked_at,omitempty"`
	Commits30d     int    `json:"commits_30d"`
	Releases90d    int    `json:"releases_90d"`
	Stars          int    `json:"stars"`
	FeatureChanges int    `json:"feature_changes"`
	Status         string `json:"status"`
}

type scenarioMatrixDocument struct {
	UpdatedAt string                  `json:"updated_at"`
	Workloads []scenarioWorkloadEntry `json:"workloads"`
}

type scenarioWorkloadEntry struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	InputTemplate   string         `json:"input_template"`
	LastBenchmarkAt string         `json:"last_benchmark_at"`
	Baseline        scenarioResult `json:"baseline"`
	Latest          scenarioResult `json:"latest"`
	Status          string         `json:"status"`
	Notes           string         `json:"notes"`
}

type scenarioResult struct {
	SuccessRate  float64 `json:"success_rate"`
	P95LatencyMS int     `json:"p95_latency_ms"`
	AvgCostUSD   float64 `json:"avg_cost_usd"`
}

type competitorTrackingDocument struct {
	UpdatedAt string                     `json:"updated_at"`
	Projects  []competitorTrackingRecord `json:"projects"`
}

type competitorTrackingRecord struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	Repo           string                    `json:"repo"`
	LastCheckedAt  string                    `json:"last_checked_at"`
	Stars          int                       `json:"stars"`
	Activity       competitorActivity        `json:"activity"`
	Release        competitorRelease         `json:"release"`
	FeatureChanges []competitorFeatureChange `json:"feature_changes"`
	Notes          string                    `json:"notes"`
}

type competitorActivity struct {
	LastCommitAt string `json:"last_commit_at"`
	Commits30d   int    `json:"commits_30d"`
}

type competitorRelease struct {
	LastReleaseAt string `json:"last_release_at"`
	Releases90d   int    `json:"releases_90d"`
}

type competitorFeatureChange struct {
	Date    string `json:"date"`
	Summary string `json:"summary"`
	Area    string `json:"area"`
}

func evaluateScenarioMatrix(cfg Config, now time.Time) (*ScenarioMatrixReport, []string, []string) {
	trimmedPath := strings.TrimSpace(cfg.ScenarioMatrixPath)
	if trimmedPath == "" {
		if cfg.RequireScenarioMatrix {
			return nil, []string{"scenario benchmark matrix path is empty"}, nil
		}
		return nil, nil, nil
	}
	resolvedPath := filepath.Clean(trimmedPath)
	payload, err := os.ReadFile(resolvedPath)
	if err != nil {
		message := fmt.Sprintf("scenario benchmark matrix is unavailable: %s", resolvedPath)
		if cfg.RequireScenarioMatrix {
			return nil, []string{message}, nil
		}
		return nil, nil, []string{message}
	}

	var doc scenarioMatrixDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		message := fmt.Sprintf("scenario benchmark matrix JSON is invalid: %s", resolvedPath)
		if cfg.RequireScenarioMatrix {
			return nil, []string{message}, nil
		}
		return nil, nil, []string{message}
	}

	report := &ScenarioMatrixReport{
		Path:      resolvedPath,
		Workloads: map[string]ScenarioWorkloadResult{},
		Gate: GateResult{
			Failures: make([]string, 0),
			Warnings: make([]string, 0),
		},
	}

	if updatedAt, ok := parseRFC3339(doc.UpdatedAt); ok {
		report.UpdatedAt = updatedAt.Format(time.RFC3339)
		report.AgeDays = int(now.Sub(updatedAt).Hours() / 24)
		if report.AgeDays > cfg.ScenarioMaxStaleDays {
			report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("scenario benchmark matrix is stale: age=%dd max=%dd", report.AgeDays, cfg.ScenarioMaxStaleDays))
		}
	} else {
		report.Gate.Failures = append(report.Gate.Failures, "scenario benchmark matrix updated_at is missing or invalid")
	}

	report.Summary.TotalWorkloads = len(doc.Workloads)
	for _, entry := range doc.Workloads {
		key := normalizeSimpleID(entry.ID)
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = key
		}
		result := ScenarioWorkloadResult{
			Name:          name,
			InputTemplate: strings.TrimSpace(entry.InputTemplate),
			SuccessRate:   entry.Latest.SuccessRate,
			P95LatencyMS:  entry.Latest.P95LatencyMS,
			AvgCostUSD:    entry.Latest.AvgCostUSD,
			DriftLevel:    classifyScenarioDrift(entry.Baseline, entry.Latest),
		}
		if result.InputTemplate == "" {
			report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("workload %s missing input_template", key))
		}
		if benchmarkAt, ok := parseRFC3339(entry.LastBenchmarkAt); ok {
			result.LastBenchmarkAt = benchmarkAt.Format(time.RFC3339)
			ageDays := int(now.Sub(benchmarkAt).Hours() / 24)
			if ageDays > cfg.ScenarioMaxStaleDays {
				report.Summary.StaleWorkloads++
				report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("workload %s benchmark is stale: age=%dd", key, ageDays))
			}
		} else {
			report.Summary.StaleWorkloads++
			report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("workload %s missing valid last_benchmark_at", key))
		}
		if entry.Latest.SuccessRate <= 0 || entry.Latest.P95LatencyMS <= 0 || entry.Latest.AvgCostUSD <= 0 {
			report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("workload %s missing latest benchmark metrics", key))
		}
		report.Workloads[key] = result
	}

	missing := missingScenarioWorkloads(report.Workloads, cfg.RequiredScenarioWorkload)
	if len(missing) > 0 {
		report.Summary.MissingWorkloads = missing
		report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("missing required workloads: %s", strings.Join(missing, ", ")))
	}
	if cfg.RequireScenarioMatrix && report.Summary.TotalWorkloads == 0 {
		report.Gate.Failures = append(report.Gate.Failures, "scenario benchmark matrix has no workload entries")
	}

	report.Gate.Passed = len(report.Gate.Failures) == 0
	if report.Gate.Passed {
		report.Summary.Status = "pass"
	} else {
		report.Summary.Status = "fail"
	}

	return report, prefixMessages("scenario matrix: ", report.Gate.Failures), prefixMessages("scenario matrix: ", report.Gate.Warnings)
}

func evaluateCompetitorTracking(cfg Config, now time.Time) (*CompetitorTrackingReport, []string, []string) {
	trimmedPath := strings.TrimSpace(cfg.CompetitorTrackingPath)
	if trimmedPath == "" {
		return nil, nil, nil
	}
	resolvedPath := filepath.Clean(trimmedPath)
	payload, err := os.ReadFile(resolvedPath)
	if err != nil {
		message := fmt.Sprintf("competitor tracking snapshot is unavailable: %s", resolvedPath)
		if cfg.RequireCompetitorTrack {
			return nil, []string{message}, nil
		}
		return nil, nil, []string{message}
	}

	var doc competitorTrackingDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		message := fmt.Sprintf("competitor tracking snapshot JSON is invalid: %s", resolvedPath)
		if cfg.RequireCompetitorTrack {
			return nil, []string{message}, nil
		}
		return nil, nil, []string{message}
	}

	report := &CompetitorTrackingReport{
		Path:     resolvedPath,
		Projects: map[string]CompetitorProjectSummary{},
		Gate: GateResult{
			Failures: make([]string, 0),
			Warnings: make([]string, 0),
		},
	}

	if updatedAt, ok := parseRFC3339(doc.UpdatedAt); ok {
		report.UpdatedAt = updatedAt.Format(time.RFC3339)
		report.AgeDays = int(now.Sub(updatedAt).Hours() / 24)
		if report.AgeDays > cfg.CompetitorMaxStaleDays {
			report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("competitor tracking snapshot is stale: age=%dd max=%dd", report.AgeDays, cfg.CompetitorMaxStaleDays))
		}
	} else {
		report.Gate.Failures = append(report.Gate.Failures, "competitor tracking updated_at is missing or invalid")
	}

	report.Summary.TotalProjects = len(doc.Projects)
	for _, item := range doc.Projects {
		key := normalizeSimpleID(item.ID)
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = key
		}
		summary := CompetitorProjectSummary{
			Name:           name,
			Repo:           strings.TrimSpace(item.Repo),
			Commits30d:     item.Activity.Commits30d,
			Releases90d:    item.Release.Releases90d,
			Stars:          item.Stars,
			FeatureChanges: len(item.FeatureChanges),
		}
		if checkedAt, ok := parseRFC3339(item.LastCheckedAt); ok {
			summary.LastCheckedAt = checkedAt.Format(time.RFC3339)
			ageDays := int(now.Sub(checkedAt).Hours() / 24)
			if ageDays > cfg.CompetitorMaxStaleDays {
				report.Summary.StaleProjects++
				report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("project %s tracking entry is stale: age=%dd", key, ageDays))
			}
		} else {
			report.Summary.StaleProjects++
			report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("project %s missing valid last_checked_at", key))
		}
		if len(item.FeatureChanges) == 0 {
			report.Summary.MissingChangeEntries++
			report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("project %s missing feature_changes entries", key))
		}
		if item.Activity.Commits30d > 0 || item.Release.Releases90d > 0 {
			report.Summary.ActiveProjects++
			summary.Status = "active"
		} else {
			summary.Status = "inactive"
		}
		report.Projects[key] = summary
	}

	if cfg.RequireCompetitorTrack && report.Summary.TotalProjects < 3 {
		report.Gate.Failures = append(report.Gate.Failures, "competitor tracking requires at least 3 projects")
	}
	if cfg.RequireCompetitorTrack && report.Summary.StaleProjects > 0 {
		report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("competitor tracking has stale project entries: %d", report.Summary.StaleProjects))
	}
	if cfg.RequireCompetitorTrack && report.Summary.MissingChangeEntries > 0 {
		report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("competitor tracking has projects without feature_changes: %d", report.Summary.MissingChangeEntries))
	}

	report.Gate.Passed = len(report.Gate.Failures) == 0
	if report.Gate.Passed {
		report.Summary.Status = "pass"
	} else {
		report.Summary.Status = "fail"
	}

	return report, prefixMessages("competitor tracking: ", report.Gate.Failures), prefixMessages("competitor tracking: ", report.Gate.Warnings)
}

func missingScenarioWorkloads(workloads map[string]ScenarioWorkloadResult, required []string) []string {
	missing := make([]string, 0)
	for _, raw := range required {
		key := normalizeSimpleID(raw)
		if _, ok := workloads[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}

func classifyScenarioDrift(baseline scenarioResult, latest scenarioResult) string {
	if baseline.SuccessRate <= 0 || baseline.P95LatencyMS <= 0 || baseline.AvgCostUSD <= 0 {
		return "unknown"
	}
	successDrop := baseline.SuccessRate - latest.SuccessRate
	latencyIncrease := float64(latest.P95LatencyMS-baseline.P95LatencyMS) / float64(baseline.P95LatencyMS)
	costIncrease := (latest.AvgCostUSD - baseline.AvgCostUSD) / baseline.AvgCostUSD

	switch {
	case successDrop >= 0.15 || latencyIncrease >= 0.40 || costIncrease >= 0.40:
		return "critical"
	case successDrop >= 0.08 || latencyIncrease >= 0.20 || costIncrease >= 0.20:
		return "high"
	case successDrop >= 0.03 || latencyIncrease >= 0.10 || costIncrease >= 0.10:
		return "medium"
	default:
		return "low"
	}
}

func normalizeSimpleID(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return "unnamed"
	}
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	return trimmed
}

func prefixMessages(prefix string, messages []string) []string {
	if len(messages) == 0 {
		return nil
	}
	result := make([]string, 0, len(messages))
	for _, message := range messages {
		result = append(result, prefix+message)
	}
	return result
}
