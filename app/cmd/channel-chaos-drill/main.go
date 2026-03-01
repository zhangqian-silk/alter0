package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"alter0/app/core/channelchaos"
)

func main() {
	matrixPath := flag.String("matrix", filepath.Join("config", "channel-chaos-matrix.json"), "path to channel chaos drill matrix json")
	outputPath := flag.String("output", filepath.Join("output", "channel-chaos", "drill-latest.json"), "path to write drill report (use - for stdout)")
	flag.Parse()

	matrix, err := channelchaos.LoadMatrix(*matrixPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos drill failed: load matrix: %v\n", err)
		os.Exit(2)
	}

	report := channelchaos.Run(context.Background(), matrix)
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "channel chaos drill failed: marshal report: %v\n", err)
		os.Exit(2)
	}
	payload = append(payload, '\n')

	if *outputPath == "-" {
		if _, err := os.Stdout.Write(payload); err != nil {
			fmt.Fprintf(os.Stderr, "channel chaos drill failed: write stdout: %v\n", err)
			os.Exit(2)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "channel chaos drill failed: create output directory: %v\n", err)
			os.Exit(2)
		}
		if err := os.WriteFile(*outputPath, payload, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "channel chaos drill failed: write report: %v\n", err)
			os.Exit(2)
		}
	}

	if !report.Passed {
		fmt.Fprintf(os.Stderr, "channel chaos drill gate failed (%d/%d failed); report=%s\n", report.FailedCount, report.ScenarioCount, *outputPath)
		for _, result := range report.Results {
			if result.Passed {
				continue
			}
			fmt.Fprintf(os.Stderr, " - %s: %s\n", result.ID, result.Failure)
		}
		os.Exit(1)
	}

	fmt.Printf("channel chaos drill passed; report=%s\n", *outputPath)
}
