package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"alter0/app/core/agent"
	"alter0/app/core/interaction/gateway"
	"alter0/app/core/orchestrator/task"
	"alter0/app/core/queue"
	"alter0/app/core/scheduler"
)

type AgentEntry struct {
	AgentID   string `json:"agent_id"`
	Workspace string `json:"workspace"`
	AgentDir  string `json:"agent_dir"`
	Executor  string `json:"executor"`
}

type StatusCollector struct {
	Gateway              *gateway.DefaultGateway
	Scheduler            *scheduler.Scheduler
	Queue                *queue.Queue
	TaskStore            *task.Store
	RepoPath             string
	CommandAuditBasePath string
	CommandAuditTailSize int
	AgentEntries         []AgentEntry
}

type commandAuditTailEntry struct {
	Timestamp string `json:"timestamp"`
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	RequestID string `json:"request_id"`
	Command   string `json:"command"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason,omitempty"`
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
	if c.Queue != nil {
		payload["queue"] = c.Queue.Stats()
	}
	if c.TaskStore != nil {
		stats, err := c.TaskStore.GlobalStats(ctx)
		if err != nil {
			payload["task"] = map[string]interface{}{"error": err.Error()}
		} else {
			payload["task"] = stats
		}
	}
	payload["executors"] = agent.ListExecutorCapabilities()
	payload["command_audit_tail"] = readCommandAuditTail(c.CommandAuditBasePath, c.CommandAuditTailSize)
	payload["git"] = gitStatus(ctx, c.RepoPath)
	if len(c.AgentEntries) > 0 {
		items := make([]AgentEntry, 0, len(c.AgentEntries))
		items = append(items, c.AgentEntries...)
		sort.Slice(items, func(i, j int) bool {
			return items[i].AgentID < items[j].AgentID
		})
		payload["agents"] = items
	}

	return payload
}

func readCommandAuditTail(basePath string, tailSize int) []commandAuditTailEntry {
	if tailSize <= 0 {
		tailSize = 10
	}
	path := strings.TrimSpace(basePath)
	if path == "" {
		path = filepath.Join("output", "audit")
	}

	dirs, err := os.ReadDir(path)
	if err != nil {
		return []commandAuditTailEntry{}
	}

	dayDirs := make([]string, 0, len(dirs))
	for _, entry := range dirs {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if _, err := time.Parse("2006-01-02", name); err != nil {
			continue
		}
		dayDirs = append(dayDirs, name)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dayDirs)))

	tail := make([]commandAuditTailEntry, 0, tailSize)
	for _, day := range dayDirs {
		entries, err := readCommandAuditDay(filepath.Join(path, day, "command_permission.jsonl"), tailSize-len(tail))
		if err != nil || len(entries) == 0 {
			continue
		}
		tail = append(tail, entries...)
		if len(tail) >= tailSize {
			break
		}
	}

	for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
		tail[i], tail[j] = tail[j], tail[i]
	}
	return tail
}

func readCommandAuditDay(path string, limit int) ([]commandAuditTailEntry, error) {
	if limit <= 0 {
		return []commandAuditTailEntry{}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lines := make([]string, 0, limit)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	items := make([]commandAuditTailEntry, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(items) < limit; i-- {
		var entry commandAuditTailEntry
		if err := json.Unmarshal([]byte(lines[i]), &entry); err != nil {
			continue
		}
		items = append(items, entry)
	}
	return items, nil
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
	if upstream, err := runGit(ctx, path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); err == nil {
		status["upstream"] = upstream
		if behind, ahead, ok := gitAheadBehind(ctx, path, upstream); ok {
			status["behind"] = behind
			status["ahead"] = ahead
		}
	}

	return status
}

func gitAheadBehind(ctx context.Context, repoPath, upstream string) (int, int, bool) {
	out, err := runGit(ctx, repoPath, "rev-list", "--left-right", "--count", upstream+"...HEAD")
	if err != nil {
		return 0, 0, false
	}
	parts := strings.Fields(out)
	if len(parts) < 2 {
		return 0, 0, false
	}
	behind, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	ahead, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return behind, ahead, true
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
