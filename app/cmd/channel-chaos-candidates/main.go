package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos candidates failed: marshal report: %v\n", err)
		os.Exit(2)
	}
	payload = append(payload, '\n')

	if *outputPath == "-" {
		if _, err := os.Stdout.Write(payload); err != nil {
			fmt.Fprintf(os.Stderr, "channel chaos candidates failed: write stdout: %v\n", err)
			os.Exit(2)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "channel chaos candidates failed: create output directory: %v\n", err)
			os.Exit(2)
		}
		if err := os.WriteFile(*outputPath, payload, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "channel chaos candidates failed: write report: %v\n", err)
			os.Exit(2)
		}
	}

	fmt.Printf("channel chaos candidates generated=%d report=%s\n", report.CandidateCount, *outputPath)
}
