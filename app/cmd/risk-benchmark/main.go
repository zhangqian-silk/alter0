package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"alter0/app/core/riskbench"
)

func main() {
	watchlistPath := flag.String("watchlist", filepath.Join("config", "risk-watchlist.json"), "path to risk watchlist json")
	runbookPath := flag.String("runbook", filepath.Join("docs", "runbooks", "risk-drift-triage.md"), "path to drift triage runbook")
	outputPath := flag.String("output", filepath.Join("output", "risk", "benchmark-latest.json"), "path to write benchmark report (use - for stdout)")
	maxStaleHours := flag.Int("max-stale-hours", 96, "max allowed watchlist age in hours")
	allowOverdue := flag.Bool("allow-overdue", false, "allow overdue items without failing gate")
	flag.Parse()

	report, err := riskbench.EvaluatePaths(*watchlistPath, *runbookPath, riskbench.Config{
		MaxStaleHours: *maxStaleHours,
		FailOnOverdue: !*allowOverdue,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "risk benchmark failed: %v\n", err)
		os.Exit(2)
	}

	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "risk benchmark failed: marshal report: %v\n", err)
		os.Exit(2)
	}
	payload = append(payload, '\n')

	if *outputPath == "-" {
		if _, err := os.Stdout.Write(payload); err != nil {
			fmt.Fprintf(os.Stderr, "risk benchmark failed: write stdout: %v\n", err)
			os.Exit(2)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "risk benchmark failed: create output directory: %v\n", err)
			os.Exit(2)
		}
		if err := os.WriteFile(*outputPath, payload, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "risk benchmark failed: write report: %v\n", err)
			os.Exit(2)
		}
	}

	if !report.Gate.Passed {
		fmt.Fprintf(os.Stderr, "risk benchmark gate failed (%d issue(s)); report=%s\n", len(report.Gate.Failures), *outputPath)
		for _, failure := range report.Gate.Failures {
			fmt.Fprintf(os.Stderr, " - %s\n", failure)
		}
		os.Exit(1)
	}

	fmt.Printf("risk benchmark gate passed; report=%s\n", *outputPath)
}
