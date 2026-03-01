package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	config "alter0/app/configs"
	executor "alter0/app/core/executor"
)

func RunPreflight(ctx context.Context, cfg config.Config, dataDir string) error {
	if err := ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	if err := checkSQLiteWritable(dataDir); err != nil {
		return fmt.Errorf("sqlite check failed: %w", err)
	}
	if _, err := executor.EnsureExecutorInstalled(ctx, cfg.Executor.Name); err != nil {
		return fmt.Errorf("executor check failed: %w", err)
	}
	return nil
}

func ValidateConfig(cfg config.Config) error {
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		return fmt.Errorf("agent.name is required")
	}
	if executor.NormalizeExecutorName(cfg.Executor.Name) == "" {
		return fmt.Errorf("executor.name is invalid: %s", cfg.Executor.Name)
	}
	if cfg.Task.RoutingTimeoutSec <= 0 {
		return fmt.Errorf("task.routing_timeout_sec must be > 0")
	}
	if cfg.Task.CloseTimeoutSec <= 0 {
		return fmt.Errorf("task.close_timeout_sec must be > 0")
	}
	if cfg.Task.GenerationTimeoutSec <= 0 {
		return fmt.Errorf("task.generation_timeout_sec must be > 0")
	}
	if cfg.Task.RoutingConfidenceThreshold <= 0 || cfg.Task.RoutingConfidenceThreshold > 1 {
		return fmt.Errorf("task.routing_confidence_threshold must be in (0,1]")
	}
	if cfg.Task.CloseConfidenceThreshold <= 0 || cfg.Task.CloseConfidenceThreshold > 1 {
		return fmt.Errorf("task.close_confidence_threshold must be in (0,1]")
	}
	if strings.TrimSpace(cfg.Task.CLIUserID) == "" {
		return fmt.Errorf("task.cli_user_id is required")
	}
	if cfg.Task.OpenTaskCandidateLimit <= 0 {
		return fmt.Errorf("task.open_task_candidate_limit must be > 0")
	}
	if cfg.Runtime.Queue.Enabled {
		if cfg.Runtime.Queue.Workers <= 0 {
			return fmt.Errorf("runtime.queue.workers must be > 0 when queue enabled")
		}
		if cfg.Runtime.Queue.Buffer <= 0 {
			return fmt.Errorf("runtime.queue.buffer must be > 0 when queue enabled")
		}
	}
	if cfg.Runtime.Maintenance.Enabled && cfg.Runtime.Maintenance.TaskMemoryRetentionDays <= 0 {
		return fmt.Errorf("runtime.maintenance.task_memory_retention_days must be > 0 when maintenance enabled")
	}
	if cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays < 0 {
		return fmt.Errorf("runtime.maintenance.task_memory_open_retention_days must be >= 0")
	}
	return nil
}

func checkSQLiteWritable(dataDir string) error {
	dir := strings.TrimSpace(dataDir)
	if dir == "" {
		return fmt.Errorf("data dir is required")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	probePath := filepath.Join(dir, ".alter0-preflight-write-check")
	f, err := os.OpenFile(probePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString("ok\n"); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Remove(probePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
