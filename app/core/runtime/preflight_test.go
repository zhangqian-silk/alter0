package runtime

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestRunPreflightFailurePaths(t *testing.T) {
	tests := []struct {
		name        string
		prepare     func(t *testing.T) (config.Config, string)
		errContains []string
	}{
		{
			name: "invalid config",
			prepare: func(t *testing.T) (config.Config, string) {
				t.Setenv("PATH", prependFakeExecutable(t, "codex"))
				cfg := validConfig("codex")
				cfg.Task.RoutingConfidenceThreshold = 2
				return cfg, t.TempDir()
			},
			errContains: []string{"config validation failed", "routing_confidence_threshold"},
		},
		{
			name: "unwritable sqlite path",
			prepare: func(t *testing.T) (config.Config, string) {
				t.Setenv("PATH", prependFakeExecutable(t, "codex"))
				cfg := validConfig("codex")
				base := t.TempDir()
				filePath := filepath.Join(base, "blocked")
				if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return cfg, filepath.Join(filePath, "db")
			},
			errContains: []string{"sqlite check failed"},
		},
		{
			name: "missing executor binary",
			prepare: func(t *testing.T) (config.Config, string) {
				t.Setenv("PATH", t.TempDir())
				return validConfig("codex"), t.TempDir()
			},
			errContains: []string{"executor check failed", "missing executable"},
		},
		{
			name: "empty sqlite data dir",
			prepare: func(t *testing.T) (config.Config, string) {
				t.Setenv("PATH", prependFakeExecutable(t, "codex"))
				return validConfig("codex"), "   "
			},
			errContains: []string{"sqlite check failed", "data dir is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, dataDir := tt.prepare(t)
			err := RunPreflight(context.Background(), cfg, dataDir)
			if err == nil {
				t.Fatalf("expected preflight failure")
			}
			for _, want := range tt.errContains {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected error to contain %q, got %q", want, err)
				}
			}
		})
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
