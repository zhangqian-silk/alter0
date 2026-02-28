package runtime

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"alter0/app/core/interaction/gateway"
	"alter0/app/core/orchestrator/task"
	"alter0/app/core/scheduler"
)

type StatusCollector struct {
	Gateway   *gateway.DefaultGateway
	Scheduler *scheduler.Scheduler
	TaskStore *task.Store
	RepoPath  string
}

func (c *StatusCollector) Snapshot(ctx context.Context) map[string]interface{} {
	payload := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if c.Gateway != nil {
		payload["gateway"] = c.Gateway.HealthStatus()
	}
	if c.Scheduler != nil {
		payload["scheduler"] = map[string]interface{}{
			"health": c.Scheduler.Health(),
			"jobs":   c.Scheduler.Snapshot(),
		}
	}
	if c.TaskStore != nil {
		stats, err := c.TaskStore.GlobalStats(ctx)
		if err != nil {
			payload["task"] = map[string]interface{}{"error": err.Error()}
		} else {
			payload["task"] = stats
		}
	}
	payload["git"] = gitStatus(ctx, c.RepoPath)

	return payload
}

func gitStatus(ctx context.Context, repoPath string) map[string]interface{} {
	status := map[string]interface{}{}
	path := strings.TrimSpace(repoPath)
	if path == "" {
		path = "."
	}
	if abs, err := filepath.Abs(path); err == nil {
		status["path"] = abs
	}

	branch, err := runGit(ctx, path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		status["error"] = err.Error()
		return status
	}
	status["branch"] = branch

	if commit, err := runGit(ctx, path, "rev-parse", "HEAD"); err == nil {
		status["commit"] = commit
	}
	if dirty, err := runGit(ctx, path, "status", "--porcelain"); err == nil {
		status["dirty"] = strings.TrimSpace(dirty) != ""
	}

	return status
}

func runGit(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
