package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"alter0/app/core/configpreflight"
)

func main() {
	configPath := flag.String("config", filepath.Join("config", "config.json"), "path to runtime config json")
	outputPath := flag.String("output", filepath.Join("output", "config", "validate-latest.json"), "path to write validation report (use - for stdout)")
	allowMissingConfig := flag.Bool("allow-missing-config", true, "allow missing config path by validating default config fallback")
	flag.Parse()

	report := configpreflight.EvaluatePath(*configPath, configpreflight.Options{AllowMissingConfig: *allowMissingConfig})

	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config preflight validation failed: marshal report: %v\n", err)
		os.Exit(2)
	}
	payload = append(payload, '\n')

	if *outputPath == "-" {
		if _, err := os.Stdout.Write(payload); err != nil {
			fmt.Fprintf(os.Stderr, "config preflight validation failed: write stdout: %v\n", err)
			os.Exit(2)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "config preflight validation failed: create output directory: %v\n", err)
			os.Exit(2)
		}
		if err := os.WriteFile(*outputPath, payload, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "config preflight validation failed: write report: %v\n", err)
			os.Exit(2)
		}
	}

	if !report.Gate.Passed {
		fmt.Fprintf(os.Stderr, "config preflight validation gate failed; report=%s\n", *outputPath)
		for _, failure := range report.Gate.Failures {
			fmt.Fprintf(os.Stderr, " - %s\n", failure)
		}
		os.Exit(1)
	}

	fmt.Printf("config preflight validation gate passed; report=%s\n", *outputPath)
}
