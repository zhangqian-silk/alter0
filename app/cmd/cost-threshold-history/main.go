package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	coreruntime "alter0/app/core/runtime"
)

const historyVersion = "2026.03-n23"

var trackedAlertCodes = []string{"session_cost_hotspot", "session_compaction_pressure"}

type statusSnapshot struct {
	Timestamp string `json:"timestamp"`
	Cost      struct {
		ThresholdGuidance map[string]interface{} `json:"threshold_guidance"`
	} `json:"cost"`
	Alerts []statusAlert `json:"alerts"`
}

type statusAlert struct {
	Code string `json:"code"`
}

type historyReport struct {
	HistoryVersion    string                 `json:"history_version"`
	GeneratedAt       string                 `json:"generated_at"`
	WindowHours       int                    `json:"window_hours"`
	Source            historySource          `json:"source"`
	ThresholdGuidance map[string]interface{} `json:"threshold_guidance"`
	Alerts            historyAlerts          `json:"alerts"`
	Archive           historyArchive         `json:"archive"`
}

type historySource struct {
	StatusTimestamp    string `json:"status_timestamp,omitempty"`
	OrchestratorLogDir string `json:"orchestrator_log_dir"`
}

type historyAlerts struct {
	Current map[string]bool `json:"current"`
	HitRate hitRateSummary  `json:"hit_rate"`
}

type hitRateSummary struct {
	Samples                   int     `json:"samples"`
	SessionCostHotspot        float64 `json:"session_cost_hotspot"`
	SessionCompactionPressure float64 `json:"session_compaction_pressure"`
}

type historyArchive struct {
	Root           string `json:"root"`
	Week           string `json:"week"`
	Snapshot       string `json:"snapshot"`
	HistorySamples int    `json:"history_samples"`
}

type archivedAlertRecord struct {
	GeneratedAt string `json:"generated_at"`
	Alerts      struct {
		Current map[string]bool `json:"current"`
	} `json:"alerts"`
}

type alertSample struct {
	Timestamp time.Time
	Current   map[string]bool
}

func main() {
	outputPath := flag.String("output", filepath.Join("output", "cost", "threshold-history-latest.json"), "path to write latest threshold-history report (use - for stdout)")
	archiveRoot := flag.String("archive-root", filepath.Join("output", "cost", "threshold-history"), "path to weekly archive root")
	orchestratorLogBase := flag.String("orchestrator-log-base", filepath.Join("output", "orchestrator"), "path to orchestrator log root")
	windowHours := flag.Int("window-hours", 24*7, "session aggregation window in hours")
	activeWindowMinutes := flag.Int("active-window-minutes", 30, "active session window in minutes")
	shareThreshold := flag.Float64("share-threshold", 0.35, "session_cost_hotspot alert share threshold")
	promptRatioThreshold := flag.Float64("prompt-ratio-threshold", 6.0, "session_compaction_pressure alert prompt/output threshold")
	minTokens := flag.Int("min-tokens", 1200, "minimum tokens to classify heavy sessions")
	maxSamples := flag.Int("max-samples", 12, "max archived samples used for alert hit-rate")
	flag.Parse()

	now := time.Now().UTC()
	snapshot, err := collectStatusSnapshot(collectorConfig{
		OrchestratorLogBase:  *orchestratorLogBase,
		Window:               time.Duration(*windowHours) * time.Hour,
		ActiveWindow:         time.Duration(*activeWindowMinutes) * time.Minute,
		ShareThreshold:       *shareThreshold,
		PromptRatioThreshold: *promptRatioThreshold,
		MinTokens:            *minTokens,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "cost threshold history failed: %v\n", err)
		os.Exit(2)
	}

	guidance := snapshot.Cost.ThresholdGuidance
	if guidance == nil {
		guidance = map[string]interface{}{
			"status": "missing",
			"reason": "status snapshot missing cost.threshold_guidance",
		}
	}
	currentAlerts := extractCurrentAlerts(snapshot.Alerts)

	historySamples, err := readHistorySamples(*archiveRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cost threshold history failed: %v\n", err)
		os.Exit(2)
	}

	hitRate := summarizeHitRate(historySamples, alertSample{
		Timestamp: now,
		Current:   currentAlerts,
	}, *maxSamples)

	week := isoWeekID(now)
	snapshotPath := filepath.Join(strings.TrimSpace(*archiveRoot), week, fmt.Sprintf("threshold-guidance-%s.json", now.Format("20060102T150405Z")))
	report := historyReport{
		HistoryVersion:    historyVersion,
		GeneratedAt:       now.Format(time.RFC3339),
		WindowHours:       *windowHours,
		Source:            historySource{StatusTimestamp: snapshot.Timestamp, OrchestratorLogDir: strings.TrimSpace(*orchestratorLogBase)},
		ThresholdGuidance: guidance,
		Alerts: historyAlerts{
			Current: currentAlerts,
			HitRate: hitRate,
		},
		Archive: historyArchive{
			Root:           strings.TrimSpace(*archiveRoot),
			Week:           week,
			Snapshot:       snapshotPath,
			HistorySamples: hitRate.Samples,
		},
	}

	if err := writeJSON(snapshotPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "cost threshold history failed: %v\n", err)
		os.Exit(2)
	}
	if *outputPath == "-" {
		if err := writeStdoutJSON(report); err != nil {
			fmt.Fprintf(os.Stderr, "cost threshold history failed: %v\n", err)
			os.Exit(2)
		}
	} else {
		if err := writeJSON(*outputPath, report); err != nil {
			fmt.Fprintf(os.Stderr, "cost threshold history failed: %v\n", err)
			os.Exit(2)
		}
	}

	fmt.Printf("cost threshold history archived; report=%s snapshot=%s\n", *outputPath, snapshotPath)
}

type collectorConfig struct {
	OrchestratorLogBase  string
	Window               time.Duration
	ActiveWindow         time.Duration
	ShareThreshold       float64
	PromptRatioThreshold float64
	MinTokens            int
}

func collectStatusSnapshot(cfg collectorConfig) (statusSnapshot, error) {
	collector := &coreruntime.StatusCollector{
		OrchestratorLogBasePath:    strings.TrimSpace(cfg.OrchestratorLogBase),
		SessionWindow:              cfg.Window,
		ActiveSessionWindow:        cfg.ActiveWindow,
		AlertSessionCostShare:      cfg.ShareThreshold,
		AlertSessionPromptOutRatio: cfg.PromptRatioThreshold,
		AlertSessionCostMinTokens:  cfg.MinTokens,
	}
	raw := collector.Snapshot(context.Background())
	payload, err := json.Marshal(raw)
	if err != nil {
		return statusSnapshot{}, fmt.Errorf("marshal runtime status snapshot: %w", err)
	}
	var snapshot statusSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return statusSnapshot{}, fmt.Errorf("decode runtime status snapshot: %w", err)
	}
	return snapshot, nil
}

func extractCurrentAlerts(alerts []statusAlert) map[string]bool {
	current := map[string]bool{}
	for _, code := range trackedAlertCodes {
		current[code] = false
	}
	for _, alert := range alerts {
		code := strings.TrimSpace(alert.Code)
		if code == "" {
			continue
		}
		if _, ok := current[code]; ok {
			current[code] = true
		}
	}
	return current
}

func readHistorySamples(root string) ([]alertSample, error) {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return nil, nil
	}
	entries := make([]alertSample, 0)
	err := filepath.WalkDir(trimmedRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(d.Name())), ".json") {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var item archivedAlertRecord
		if err := json.Unmarshal(payload, &item); err != nil {
			return nil
		}
		timestamp, ok := parseRFC3339(item.GeneratedAt)
		if !ok {
			return nil
		}
		current := map[string]bool{}
		for _, code := range trackedAlertCodes {
			current[code] = false
		}
		for code, triggered := range item.Alerts.Current {
			if _, ok := current[code]; ok {
				current[code] = triggered
			}
		}
		entries = append(entries, alertSample{Timestamp: timestamp, Current: current})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("scan archive root %s: %w", trimmedRoot, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	return entries, nil
}

func summarizeHitRate(history []alertSample, current alertSample, maxSamples int) hitRateSummary {
	samples := append([]alertSample{}, history...)
	samples = append(samples, current)
	if maxSamples > 0 && len(samples) > maxSamples {
		samples = samples[len(samples)-maxSamples:]
	}
	if len(samples) == 0 {
		return hitRateSummary{}
	}

	hotspotHits := 0
	compactionHits := 0
	for _, sample := range samples {
		if sample.Current["session_cost_hotspot"] {
			hotspotHits++
		}
		if sample.Current["session_compaction_pressure"] {
			compactionHits++
		}
	}

	total := float64(len(samples))
	return hitRateSummary{
		Samples:                   len(samples),
		SessionCostHotspot:        roundFloat(float64(hotspotHits)/total, 3),
		SessionCompactionPressure: roundFloat(float64(compactionHits)/total, 3),
	}
}

func writeJSON(path string, payload interface{}) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeStdoutJSON(payload interface{}) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

func parseRFC3339(raw string) (time.Time, bool) {
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

func isoWeekID(now time.Time) string {
	year, week := now.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

func roundFloat(value float64, places int) float64 {
	if places < 0 {
		return value
	}
	factor := 1.0
	for i := 0; i < places; i++ {
		factor *= 10
	}
	return float64(int(value*factor+0.5)) / factor
}
