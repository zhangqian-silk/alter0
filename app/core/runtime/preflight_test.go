package runtime

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	config "alter0/app/configs"
)

func TestRunPreflightPasses(t *testing.T) {
	t.Setenv("PATH", prependFakeExecutable(t, "codex"))
	dir := filepath.Join(t.TempDir(), "db")
	cfg := validConfig("codex")

	if err := RunPreflight(context.Background(), cfg, dir); err != nil {
		t.Fatalf("expected preflight success, got %v", err)
	}
}

func TestRunPreflightRejectsInvalidConfig(t *testing.T) {
	t.Setenv("PATH", prependFakeExecutable(t, "codex"))
	cfg := validConfig("codex")
	cfg.Task.RoutingConfidenceThreshold = 2

	err := RunPreflight(context.Background(), cfg, t.TempDir())
	if err == nil {
		t.Fatalf("expected preflight failure for invalid config")
	}
}

func TestRunPreflightRejectsUnwritableSQLitePath(t *testing.T) {
	t.Setenv("PATH", prependFakeExecutable(t, "codex"))
	cfg := validConfig("codex")
	base := t.TempDir()
	filePath := filepath.Join(base, "blocked")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := RunPreflight(context.Background(), cfg, filepath.Join(filePath, "db"))
	if err == nil {
		t.Fatalf("expected preflight failure for unwritable sqlite path")
	}
}

func TestRunPreflightRejectsMissingExecutorBinary(t *testing.T) {
	cfg := validConfig("codex")
	t.Setenv("PATH", t.TempDir())

	err := RunPreflight(context.Background(), cfg, t.TempDir())
	if err == nil {
		t.Fatalf("expected preflight failure for missing executor")
	}
}

func validConfig(executor string) config.Config {
	return config.Config{
		Agent: config.AgentConfig{Name: "Alter0"},
		Executor: config.ExecutorConfig{
			Name: executor,
		},
		Task: config.TaskConfig{
			RoutingTimeoutSec:          15,
			CloseTimeoutSec:            10,
			GenerationTimeoutSec:       60,
			RoutingConfidenceThreshold: 0.55,
			CloseConfidenceThreshold:   0.7,
			CLIUserID:                  "local_user",
			OpenTaskCandidateLimit:     8,
		},
		Runtime: config.RuntimeConfig{
			Maintenance: config.MaintenanceConfig{
				Enabled:                     true,
				TaskMemoryPruneIntervalSec:  3600,
				TaskMemoryPruneTimeoutSec:   15,
				TaskMemoryRetentionDays:     30,
				TaskMemoryOpenRetentionDays: 0,
			},
			Queue: config.ExecutionQueueConfig{
				Enabled: true,
				Workers: 1,
				Buffer:  32,
			},
		},
		Security: config.SecurityConfig{AdminUserIDs: []string{"local_user"}},
	}
}

func prependFakeExecutable(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		path += ".bat"
		content = "@echo off\r\nexit /b 0\r\n"
	}
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}
	return dir + string(os.PathListSeparator) + os.Getenv("PATH")
}
