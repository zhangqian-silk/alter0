package channelchaos

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const CalibrationVersion = "2026.03-n33"

type CalibrationOptions struct {
	Now                    time.Time
	Window                 time.Duration
	CandidateArchiveRoot   string
	MatrixPath             string
	ThresholdHistoryRoot   string
	ThresholdReconcileRoot string
}

type CalibrationArchive struct {
	Root     string `json:"root,omitempty"`
	Week     string `json:"week,omitempty"`
	Snapshot string `json:"snapshot,omitempty"`
}

type CalibrationReport struct {
	CalibrationVersion string               `json:"calibration_version"`
	GeneratedAt        string               `json:"generated_at"`
	WindowDays         int                  `json:"window_days"`
	Candidate          CandidateCalibration `json:"candidate"`
	Threshold          ThresholdCalibration `json:"threshold"`
	FalsePositive      FalsePositiveSummary `json:"false_positive"`
	Archive            CalibrationArchive   `json:"archive,omitempty"`
	Notes              []string             `json:"notes,omitempty"`
}

type CandidateCalibration struct {
	Reports                int                           `json:"reports"`
	UniqueCandidates       int                           `json:"unique_candidates"`
	MatrixMapped           int                           `json:"matrix_mapped"`
	MatrixScenarios        int                           `json:"matrix_scenarios"`
	TaggedScenarios        int                           `json:"tagged_scenarios"`
	TagCoverage            float64                       `json:"tag_coverage"`
	AdoptedCandidates      int                           `json:"adopted_candidates"`
	AdoptionRate           float64                       `json:"adoption_rate"`
	MatrixUnseenCandidates int                           `json:"matrix_unseen_candidates"`
	AdoptionByChannel      []CandidateChannelCalibration `json:"adoption_by_channel,omitempty"`
	MissingScenarioTags    []string                      `json:"missing_scenario_tags,omitempty"`
	PendingCandidates      []string                      `json:"pending_candidates,omitempty"`
}

type CandidateChannelCalibration struct {
	Channel string `json:"channel"`
	Seen    int    `json:"seen"`
	Adopted int    `json:"adopted"`
}

type ThresholdCalibration struct {
	Samples     int     `json:"samples"`
	ReadyRate   float64 `json:"ready_rate"`
	AppliedRate float64 `json:"applied_rate"`
}

type FalsePositiveSummary struct {
	Samples         int     `json:"samples"`
	BaselineHitRate float64 `json:"baseline_hit_rate"`
	CurrentHitRate  float64 `json:"current_hit_rate"`
	ReductionRate   float64 `json:"reduction_rate"`
	Status          string  `json:"status"`
}

type thresholdHistorySample struct {
	Timestamp time.Time
	HitRate   float64
}

type thresholdReconcileSample struct {
	Timestamp time.Time
	Status    string
	Applied   bool
}

func BuildCalibrationReport(opts CalibrationOptions) (CalibrationReport, error) {
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	window := opts.Window
	if window <= 0 {
		window = 14 * 24 * time.Hour
	}
	since := now.Add(-window)

	report := CalibrationReport{
		CalibrationVersion: CalibrationVersion,
		GeneratedAt:        now.Format(time.RFC3339),
		WindowDays:         int(window / (24 * time.Hour)),
		Candidate: CandidateCalibration{
			PendingCandidates: []string{},
		},
		Threshold:     ThresholdCalibration{},
		FalsePositive: FalsePositiveSummary{Status: "no_data"},
		Notes:         []string{},
	}

	matrix, err := LoadMatrix(opts.MatrixPath)
	if err != nil {
		return report, fmt.Errorf("load matrix: %w", err)
	}
	matrixMapped := map[string]struct{}{}
	missingScenarioTags := make([]string, 0)
	for _, scenario := range matrix.Scenarios {
		report.Candidate.MatrixScenarios++
		key := strings.TrimSpace(scenario.SourceCandidate)
		if key == "" {
			id := strings.TrimSpace(scenario.ID)
			if id == "" {
				id = "(unnamed)"
			}
			missingScenarioTags = append(missingScenarioTags, id)
			continue
		}
		report.Candidate.TaggedScenarios++
		matrixMapped[key] = struct{}{}
	}
	sort.Strings(missingScenarioTags)
	report.Candidate.MatrixMapped = len(matrixMapped)
	report.Candidate.MissingScenarioTags = missingScenarioTags
	if report.Candidate.MatrixScenarios > 0 {
		report.Candidate.TagCoverage = roundCalibrationFloat(float64(report.Candidate.TaggedScenarios)/float64(report.Candidate.MatrixScenarios), 3)
	}
	if len(missingScenarioTags) > 0 {
		report.Notes = append(report.Notes, fmt.Sprintf("matrix has %d/%d scenarios without source_candidate tags", len(missingScenarioTags), report.Candidate.MatrixScenarios))
	}
	if report.Candidate.MatrixMapped == 0 {
		report.Notes = append(report.Notes, "matrix has no source_candidate tags; adoption rate will stay 0 until scenarios link to sampled candidates")
	}

	candidateReports, candidateSet, err := readCandidateArchive(opts.CandidateArchiveRoot, since)
	if err != nil {
		return report, fmt.Errorf("read candidate archive: %w", err)
	}
	report.Candidate.Reports = candidateReports
	report.Candidate.UniqueCandidates = len(candidateSet)

	adopted := 0
	pending := make([]string, 0)
	channelStats := map[string]*CandidateChannelCalibration{}
	for key := range candidateSet {
		channel := parseSourceCandidateChannel(key)
		stats := channelStats[channel]
		if stats == nil {
			stats = &CandidateChannelCalibration{Channel: channel}
			channelStats[channel] = stats
		}
		stats.Seen++

		if _, ok := matrixMapped[key]; ok {
			adopted++
			stats.Adopted++
			continue
		}
		pending = append(pending, key)
	}
	sort.Strings(pending)
	report.Candidate.AdoptedCandidates = adopted
	report.Candidate.PendingCandidates = pending
	if report.Candidate.UniqueCandidates > 0 {
		report.Candidate.AdoptionRate = roundCalibrationFloat(float64(adopted)/float64(report.Candidate.UniqueCandidates), 3)
	}

	matrixUnseen := 0
	for key := range matrixMapped {
		if _, ok := candidateSet[key]; ok {
			continue
		}
		matrixUnseen++
	}
	report.Candidate.MatrixUnseenCandidates = matrixUnseen
	if report.Candidate.MatrixMapped > 0 && report.Candidate.UniqueCandidates == 0 {
		report.Notes = append(report.Notes, "matrix has source_candidate tags but no sampled candidates in selected window")
	}

	if len(channelStats) > 0 {
		channels := make([]string, 0, len(channelStats))
		for channel := range channelStats {
			channels = append(channels, channel)
		}
		sort.Strings(channels)
		report.Candidate.AdoptionByChannel = make([]CandidateChannelCalibration, 0, len(channels))
		for _, channel := range channels {
			report.Candidate.AdoptionByChannel = append(report.Candidate.AdoptionByChannel, *channelStats[channel])
		}
	}

	historySamples, err := readThresholdHistorySamples(opts.ThresholdHistoryRoot, since)
	if err != nil {
		return report, fmt.Errorf("read threshold history: %w", err)
	}
	report.FalsePositive = summarizeFalsePositive(historySamples)

	reconcileSamples, err := readThresholdReconcileSamples(opts.ThresholdReconcileRoot, since)
	if err != nil {
		return report, fmt.Errorf("read threshold reconcile: %w", err)
	}
	report.Threshold = summarizeThresholdCadence(reconcileSamples)
	if report.Threshold.Samples == 0 {
		report.Notes = append(report.Notes, "no threshold-reconcile samples found in selected window")
	}
	if report.Candidate.Reports == 0 {
		report.Notes = append(report.Notes, "no candidate archive samples found in selected window")
	}
	if report.FalsePositive.Samples == 0 {
		report.Notes = append(report.Notes, "no threshold-history samples found in selected window")
	}

	if len(report.Notes) == 0 {
		report.Notes = nil
	}
	return report, nil
}

func readCandidateArchive(root string, since time.Time) (int, map[string]struct{}, error) {
	files, err := collectJSONFiles(root)
	if err != nil {
		return 0, nil, err
	}

	reports := 0
	keys := map[string]struct{}{}
	for _, path := range files {
		payload, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var report CandidateReport
		if err := json.Unmarshal(payload, &report); err != nil {
			continue
		}
		timestamp, ok := parseCalibrationTimestamp(report.GeneratedAt)
		if !ok || timestamp.Before(since) {
			continue
		}
		reports++
		for _, candidate := range report.Candidates {
			key := strings.TrimSpace(candidate.SourceCandidate)
			if key == "" {
				key = strings.TrimSpace(candidate.ID)
			}
			if key == "" {
				continue
			}
			keys[key] = struct{}{}
		}
	}
	return reports, keys, nil
}

func readThresholdHistorySamples(root string, since time.Time) ([]thresholdHistorySample, error) {
	type thresholdHistoryDocument struct {
		GeneratedAt string `json:"generated_at"`
		Alerts      struct {
			HitRate struct {
				SessionCostHotspot        float64 `json:"session_cost_hotspot"`
				SessionCompactionPressure float64 `json:"session_compaction_pressure"`
			} `json:"hit_rate"`
		} `json:"alerts"`
	}

	files, err := collectJSONFiles(root)
	if err != nil {
		return nil, err
	}
	dedupe := map[string]thresholdHistorySample{}
	for _, path := range files {
		payload, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc thresholdHistoryDocument
		if err := json.Unmarshal(payload, &doc); err != nil {
			continue
		}
		timestamp, ok := parseCalibrationTimestamp(doc.GeneratedAt)
		if !ok || timestamp.Before(since) {
			continue
		}
		hitRate := (doc.Alerts.HitRate.SessionCostHotspot + doc.Alerts.HitRate.SessionCompactionPressure) / 2
		key := timestamp.Format(time.RFC3339Nano)
		dedupe[key] = thresholdHistorySample{Timestamp: timestamp, HitRate: hitRate}
	}
	samples := make([]thresholdHistorySample, 0, len(dedupe))
	for _, item := range dedupe {
		samples = append(samples, item)
	}
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].Timestamp.Before(samples[j].Timestamp)
	})
	return samples, nil
}

func readThresholdReconcileSamples(root string, since time.Time) ([]thresholdReconcileSample, error) {
	type thresholdReconcileDocument struct {
		GeneratedAt string `json:"generated_at"`
		Applied     bool   `json:"applied"`
		Plan        struct {
			Status string `json:"status"`
		} `json:"plan"`
	}

	files, err := collectJSONFiles(root)
	if err != nil {
		return nil, err
	}
	dedupe := map[string]thresholdReconcileSample{}
	for _, path := range files {
		payload, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc thresholdReconcileDocument
		if err := json.Unmarshal(payload, &doc); err != nil {
			continue
		}
		timestamp, ok := parseCalibrationTimestamp(doc.GeneratedAt)
		if !ok || timestamp.Before(since) {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(doc.Plan.Status))
		if status == "" {
			status = "unknown"
		}
		key := timestamp.Format(time.RFC3339Nano)
		dedupe[key] = thresholdReconcileSample{Timestamp: timestamp, Status: status, Applied: doc.Applied}
	}
	samples := make([]thresholdReconcileSample, 0, len(dedupe))
	for _, item := range dedupe {
		samples = append(samples, item)
	}
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].Timestamp.Before(samples[j].Timestamp)
	})
	return samples, nil
}

func summarizeThresholdCadence(samples []thresholdReconcileSample) ThresholdCalibration {
	if len(samples) == 0 {
		return ThresholdCalibration{}
	}
	ready := 0
	applied := 0
	for _, sample := range samples {
		if sample.Status == "ready" || sample.Status == "applied" {
			ready++
		}
		if sample.Applied {
			applied++
		}
	}
	total := float64(len(samples))
	return ThresholdCalibration{
		Samples:     len(samples),
		ReadyRate:   roundCalibrationFloat(float64(ready)/total, 3),
		AppliedRate: roundCalibrationFloat(float64(applied)/total, 3),
	}
}

func summarizeFalsePositive(samples []thresholdHistorySample) FalsePositiveSummary {
	if len(samples) == 0 {
		return FalsePositiveSummary{Status: "no_data"}
	}
	baseline := samples[0].HitRate
	current := samples[len(samples)-1].HitRate
	summary := FalsePositiveSummary{
		Samples:         len(samples),
		BaselineHitRate: roundCalibrationFloat(baseline, 3),
		CurrentHitRate:  roundCalibrationFloat(current, 3),
		ReductionRate:   0,
		Status:          "stable",
	}
	if baseline == 0 {
		if current == 0 {
			summary.Status = "stable_zero"
			return summary
		}
		summary.Status = "increased_from_zero"
		summary.ReductionRate = -1
		return summary
	}
	summary.ReductionRate = roundCalibrationFloat((baseline-current)/baseline, 3)
	if current < baseline {
		summary.Status = "improved"
	} else if current > baseline {
		summary.Status = "regressed"
	}
	return summary
}

func collectJSONFiles(root string) ([]string, error) {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return nil, nil
	}
	files := make([]string, 0)
	err := filepath.WalkDir(trimmed, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(strings.TrimSpace(d.Name()))
		if !strings.HasSuffix(name, ".json") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("scan %s: %w", trimmed, err)
	}
	return files, nil
}

func parseCalibrationTimestamp(raw string) (time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

func parseSourceCandidateChannel(key string) string {
	trimmed := strings.TrimSpace(strings.ToLower(key))
	if trimmed == "" {
		return "unknown"
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) >= 2 && parts[0] == "trace" {
		channel := strings.TrimSpace(parts[1])
		if channel != "" {
			return channel
		}
	}
	return "unknown"
}

func roundCalibrationFloat(value float64, places int) float64 {
	if places < 0 {
		return value
	}
	factor := 1.0
	for i := 0; i < places; i++ {
		factor *= 10
	}
	if value >= 0 {
		return float64(int(value*factor+0.5)) / factor
	}
	return float64(int(value*factor-0.5)) / factor
}
