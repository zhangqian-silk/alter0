package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"alter0/app/core/channelchaos"
)

func main() {
	candidateArchiveRoot := flag.String("candidate-archive-root", filepath.Join("output", "channel-chaos", "candidates"), "path to channel-chaos candidate archive root")
	matrixPath := flag.String("matrix", filepath.Join("config", "channel-chaos-matrix.json"), "path to channel-chaos matrix json")
	thresholdHistoryRoot := flag.String("threshold-history-root", filepath.Join("output", "cost", "threshold-history"), "path to threshold-history archive root")
	thresholdReconcileRoot := flag.String("threshold-reconcile-root", filepath.Join("output", "cost", "threshold-reconcile"), "path to threshold-reconcile archive root")
	windowText := flag.String("window", "336h", "lookback window (Go duration format)")
	outputPath := flag.String("output", filepath.Join("output", "channel-chaos", "calibration-latest.json"), "path to write calibration report (use - for stdout)")
	archiveRoot := flag.String("archive-root", filepath.Join("output", "channel-chaos", "calibration"), "path to weekly calibration archive root")
	flag.Parse()

	window, err := time.ParseDuration(*windowText)
	if err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos calibration failed: invalid --window: %v\n", err)
		os.Exit(2)
	}

	report, err := channelchaos.BuildCalibrationReport(channelchaos.CalibrationOptions{
		Window:                 window,
		CandidateArchiveRoot:   *candidateArchiveRoot,
		MatrixPath:             *matrixPath,
		ThresholdHistoryRoot:   *thresholdHistoryRoot,
		ThresholdReconcileRoot: *thresholdReconcileRoot,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos calibration failed: %v\n", err)
		os.Exit(2)
	}

	if root := strings.TrimSpace(*archiveRoot); root != "" {
		now, ok := parseTimestamp(report.GeneratedAt)
		if !ok {
			now = time.Now().UTC()
		}
		week := isoWeekID(now)
		snapshotPath := filepath.Join(root, week, fmt.Sprintf("calibration-%s.json", now.Format("20060102T150405Z")))
		report.Archive = channelchaos.CalibrationArchive{
			Root:     root,
			Week:     week,
			Snapshot: snapshotPath,
		}
		if err := writeReport(snapshotPath, report); err != nil {
			fmt.Fprintf(os.Stderr, "channel chaos calibration failed: %v\n", err)
			os.Exit(2)
		}
	}

	if err := writeReport(*outputPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos calibration failed: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("channel chaos calibration finished; adoption_rate=%.3f reduction_rate=%.3f report=%s\n", report.Candidate.AdoptionRate, report.FalsePositive.ReductionRate, *outputPath)
}

func writeReport(path string, report channelchaos.CalibrationReport) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	payload = append(payload, '\n')
	if path == "-" {
		_, err := os.Stdout.Write(payload)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(path, payload, 0644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func parseTimestamp(raw string) (time.Time, bool) {
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw)); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw)); err == nil {
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

func isoWeekID(now time.Time) string {
	year, week := now.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}
