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
	scenarioPath := flag.String("scenario-matrix", filepath.Join("config", "scenario-benchmark-matrix.json"), "path to scenario benchmark matrix json")
	competitorPath := flag.String("competitor-tracking", filepath.Join("config", "competitor-tracking.json"), "path to competitor tracking snapshot json")
	thresholdHistoryPath := flag.String("threshold-history", filepath.Join("output", "cost", "threshold-history-latest.json"), "path to cost threshold history snapshot json")
	outputPath := flag.String("output", filepath.Join("output", "risk", "benchmark-latest.json"), "path to write benchmark report (use - for stdout)")
	maxStaleHours := flag.Int("max-stale-hours", 96, "max allowed watchlist age in hours")
	scenarioMaxStaleDays := flag.Int("scenario-max-stale-days", 45, "max allowed scenario matrix age in days")
	competitorMaxStaleDays := flag.Int("competitor-max-stale-days", 31, "max allowed competitor tracking age in days")
	thresholdMaxStaleDays := flag.Int("threshold-max-stale-days", 8, "max allowed cost threshold history age in days")
	allowOverdue := flag.Bool("allow-overdue", false, "allow overdue watchlist items without failing gate")
	skipScenario := flag.Bool("skip-scenario", false, "skip scenario benchmark matrix validation")
	skipCompetitor := flag.Bool("skip-competitor", false, "skip competitor tracking validation")
	skipThresholdHistory := flag.Bool("skip-threshold-history", false, "skip cost threshold history validation")
	flag.Parse()

	report, err := riskbench.EvaluatePaths(*watchlistPath, *runbookPath, riskbench.Config{
		MaxStaleHours:           *maxStaleHours,
		FailOnOverdue:           !*allowOverdue,
		ScenarioMatrixPath:      *scenarioPath,
		CompetitorTrackingPath:  *competitorPath,
		ThresholdHistoryPath:    *thresholdHistoryPath,
		ScenarioMaxStaleDays:    *scenarioMaxStaleDays,
		CompetitorMaxStaleDays:  *competitorMaxStaleDays,
		ThresholdMaxStaleDays:   *thresholdMaxStaleDays,
		RequireScenarioMatrix:   !*skipScenario,
		RequireCompetitorTrack:  !*skipCompetitor,
		RequireThresholdHistory: !*skipThresholdHistory,
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
