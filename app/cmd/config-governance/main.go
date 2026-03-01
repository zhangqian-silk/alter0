package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"alter0/app/core/configaudit"
)

func main() {
	configPath := flag.String("config", filepath.Join("config", "config.json"), "path to runtime config json")
	outputPath := flag.String("output", filepath.Join("output", "config", "governance-latest.json"), "path to write governance report (use - for stdout)")
	flag.Parse()

	report, err := configaudit.EvaluatePath(*configPath, configaudit.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "config governance audit failed: %v\n", err)
		os.Exit(1)
	}

	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal config governance report failed: %v\n", err)
		os.Exit(1)
	}

	if *outputPath == "-" {
		if _, err := fmt.Fprintln(os.Stdout, string(payload)); err != nil {
			fmt.Fprintf(os.Stderr, "write stdout failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "prepare output directory failed: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*outputPath, append(payload, '\n'), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write report failed: %v\n", err)
			os.Exit(1)
		}
	}

	if !report.Gate.Passed {
		fmt.Fprintf(os.Stderr, "config governance gate failed; report=%s\n", *outputPath)
		os.Exit(1)
	}

	fmt.Printf("config governance gate passed; report=%s\n", *outputPath)
}
