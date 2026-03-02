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
	traceBase := flag.String("trace-base", filepath.Join("output", "trace"), "path to gateway trace base directory")
	windowText := flag.String("window", "24h", "trace lookback window (Go duration format, e.g. 6h, 30m)")
	minErrors := flag.Int("min-errors", 2, "minimum channel error events to emit a candidate")
	maxCandidates := flag.Int("max-candidates", 5, "maximum number of candidates to emit")
	maxEvents := flag.Int("max-events", 8, "maximum number of trace events retained per candidate scenario")
	outputPath := flag.String("output", filepath.Join("output", "channel-chaos", "candidates-latest.json"), "path to write candidate report (use - for stdout)")
	archiveRoot := flag.String("archive-root", filepath.Join("output", "channel-chaos", "candidates"), "path to weekly archive root")
	flag.Parse()

	window, err := time.ParseDuration(*windowText)
	if err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos candidates failed: invalid --window: %v\n", err)
		os.Exit(2)
	}

	report, err := channelchaos.BuildCandidatesFromTrace(*traceBase, channelchaos.CandidateOptions{
		Window:               window,
		MinErrorEvents:       *minErrors,
		MaxCandidates:        *maxCandidates,
		MaxEventsPerScenario: *maxEvents,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos candidates failed: %v\n", err)
		os.Exit(2)
	}

	if err := writeReport(*outputPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos candidates failed: %v\n", err)
		os.Exit(2)
	}

	if root := strings.TrimSpace(*archiveRoot); root != "" {
		now, ok := parseTimestamp(report.GeneratedAt)
		if !ok {
			now = time.Now().UTC()
		}
		snapshot := filepath.Join(root, isoWeekID(now), fmt.Sprintf("candidates-%s.json", now.Format("20060102T150405Z")))
		if err := writeReport(snapshot, report); err != nil {
			fmt.Fprintf(os.Stderr, "channel chaos candidates failed: %v\n", err)
			os.Exit(2)
		}
	}

	fmt.Printf("channel chaos candidates generated=%d report=%s\n", report.CandidateCount, *outputPath)
}

func writeReport(path string, report channelchaos.CandidateReport) error {
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
