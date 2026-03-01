package configaudit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	config "alter0/app/configs"
)

func TestEvaluatePathPassesWithDefaultConfig(t *testing.T) {
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	path := writeConfigFixture(t, config.DefaultConfig())

	report, err := EvaluatePath(path, Options{Now: now})
	if err != nil {
		t.Fatalf("EvaluatePath returned error: %v", err)
	}
	if !report.Gate.Passed {
		t.Fatalf("expected gate pass, got failures=%v warnings=%v", report.Gate.Failures, report.Gate.Warnings)
	}
	if report.Summary.TotalParameters == 0 {
		t.Fatalf("expected non-empty parameter coverage")
	}
	if report.Domains["agents"].TotalParameters == 0 {
		t.Fatalf("expected agents domain coverage")
	}
}

func TestEvaluatePathFailsWithoutRequiredConfirmTools(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Security.Tools.RequireConfirm = []string{"browser"}
	path := writeConfigFixture(t, cfg)

	report, err := EvaluatePath(path, Options{Now: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("EvaluatePath returned error: %v", err)
	}
	if report.Gate.Passed {
		t.Fatalf("expected gate fail")
	}
	if !containsSubstring(report.Gate.Failures, "security.tools.require_confirm") {
		t.Fatalf("expected require_confirm failure, got %v", report.Gate.Failures)
	}
}

func TestEvaluatePathFailsWhenTelegramEnabledWithoutToken(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.BotToken = ""
	path := writeConfigFixture(t, cfg)

	report, err := EvaluatePath(path, Options{Now: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("EvaluatePath returned error: %v", err)
	}
	if report.Gate.Passed {
		t.Fatalf("expected gate fail")
	}
	if !containsSubstring(report.Gate.Failures, "channels.telegram.enabled") {
		t.Fatalf("expected telegram binding failure, got %v", report.Gate.Failures)
	}
}

func TestEvaluatePathWarnsOnHighWorkerCount(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.Queue.Workers = 12
	path := writeConfigFixture(t, cfg)

	report, err := EvaluatePath(path, Options{Now: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("EvaluatePath returned error: %v", err)
	}
	if !report.Gate.Passed {
		t.Fatalf("expected gate pass with warning, got failures=%v", report.Gate.Failures)
	}
	if !containsSubstring(report.Gate.Warnings, "runtime.queue.workers") {
		t.Fatalf("expected worker warning, got %v", report.Gate.Warnings)
	}
}

func writeConfigFixture(t *testing.T, cfg config.Config) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	payload, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config fixture: %v", err)
	}
	if err := os.WriteFile(path, payload, 0644); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return path
}

func containsSubstring(items []string, target string) bool {
	for _, item := range items {
		if strings.Contains(item, target) {
			return true
		}
	}
	return false
}
