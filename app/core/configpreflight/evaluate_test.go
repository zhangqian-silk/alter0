package configpreflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluatePathPassesWithExistingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"executor":{"name":"codex"}}`), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	report := EvaluatePath(configPath, Options{AllowMissingConfig: false})
	if report.Status != "ok" {
		t.Fatalf("expected status ok, got %#v", report.Status)
	}
	if !report.Gate.Passed {
		t.Fatalf("expected gate passed, got %#v", report.Gate)
	}
	if !report.ConfigExists {
		t.Fatalf("expected config exists true, got %#v", report.ConfigExists)
	}
	if report.UsedDefaultConfig {
		t.Fatalf("expected used_default_config false, got %#v", report.UsedDefaultConfig)
	}
}

func TestEvaluatePathUsesDefaultWhenConfigMissing(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing-config.json")

	report := EvaluatePath(configPath, Options{AllowMissingConfig: true})
	if report.Status != "ok" {
		t.Fatalf("expected status ok, got %#v", report.Status)
	}
	if !report.Gate.Passed {
		t.Fatalf("expected gate passed, got %#v", report.Gate)
	}
	if report.ConfigExists {
		t.Fatalf("expected config_exists false, got %#v", report.ConfigExists)
	}
	if !report.UsedDefaultConfig {
		t.Fatalf("expected used_default_config true, got %#v", report.UsedDefaultConfig)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected missing config path to stay absent, got err=%v", err)
	}
}

func TestEvaluatePathFailsWhenConfigMissingAndDisallowed(t *testing.T) {
	report := EvaluatePath(filepath.Join(t.TempDir(), "missing-config.json"), Options{AllowMissingConfig: false})
	if report.Status != "failed" {
		t.Fatalf("expected status failed, got %#v", report.Status)
	}
	if report.Gate.Passed {
		t.Fatalf("expected gate failed, got %#v", report.Gate)
	}
	if len(report.Gate.Failures) == 0 {
		t.Fatalf("expected at least one failure, got %#v", report.Gate)
	}
	if !strings.Contains(report.Gate.Failures[0], "config file not found") {
		t.Fatalf("expected missing-config failure, got %#v", report.Gate.Failures)
	}
}

func TestEvaluatePathFailsOnInvalidConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(configPath, []byte(`{"executor":{"name":"invalid_executor"}}`), 0644); err != nil {
		t.Fatalf("write invalid config failed: %v", err)
	}

	report := EvaluatePath(configPath, Options{AllowMissingConfig: false})
	if report.Status != "failed" {
		t.Fatalf("expected status failed, got %#v", report.Status)
	}
	if report.Gate.Passed {
		t.Fatalf("expected gate failed, got %#v", report.Gate)
	}
	if len(report.Gate.Failures) == 0 {
		t.Fatalf("expected failure list, got %#v", report.Gate)
	}
	if !strings.Contains(report.Gate.Failures[0], "executor.name is invalid") {
		t.Fatalf("expected executor validation failure, got %#v", report.Gate.Failures)
	}
}
