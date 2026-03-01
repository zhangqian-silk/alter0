package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"alter0/app/core/executor"
	orctask "alter0/app/core/orchestrator/task"
	toolruntime "alter0/app/core/runtime/tools"
	schedulesvc "alter0/app/core/service/schedule"
	"alter0/app/pkg/queue"
	"alter0/app/pkg/scheduler"
)

type AgentEntry struct {
	AgentID   string `json:"agent_id"`
	Workspace string `json:"workspace"`
	AgentDir  string `json:"agent_dir"`
	Executor  string `json:"executor"`
}

type StatusCollector struct {
	GatewayStatusProvider   func() interface{}
	Scheduler               *scheduler.Scheduler
	Queue                   *queue.Queue
	TaskStore               *orctask.Store
	ScheduleService         *schedulesvc.Service
	RepoPath                string
	CommandAuditBasePath    string
	CommandAuditTailSize    int
	OrchestratorLogBasePath string
	SessionWindow           time.Duration
	ActiveSessionWindow     time.Duration
	CostInputPer1K          float64
	CostOutputPer1K         float64
	AgentEntries            []AgentEntry
	ToolRuntime             *toolruntime.Runtime
	GatewayTraceBasePath    string
	GatewayTraceWindow      time.Duration
	AlertRetryStormWindow   time.Duration
	AlertRetryStormMinimum  int
	AlertQueueBacklogRatio  float64
	AlertQueueBacklogDepth  int
	AlertExecutorStrictMode bool
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

type orchestratorLogEntry struct {
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"session_id"`
	Stage       string `json:"stage"`
	Executor    string `json:"executor"`
	Status      string `json:"status"`
	PromptChars int    `json:"prompt_chars"`
	OutputChars int    `json:"output_chars"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
}

type runtimeUsageSummary struct {
	Sessions map[string]interface{}
	Subagent map[string]interface{}
	Cost     map[string]interface{}
}

type gatewayTraceEntry struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id"`
	MessageID string `json:"message_id"`
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id"`
	AgentID   string `json:"agent_id"`
	Event     string `json:"event"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
}

type gatewayTraceSummary struct {
	WindowMinutes int
	TotalEvents   int
	ErrorEvents   int
	ByEvent       map[string]int
	ErrorByEvent  map[string]int
	ByChannel     map[string]int
	ErrorChannels map[string]int
	LatestAt      string
}

type runtimeAlert struct {
	Code      string                 `json:"code"`
	Severity  string                 `json:"severity"`
	Source    string                 `json:"source"`
	Message   string                 `json:"message"`
	Detected  string                 `json:"detected_at"`
	Telemetry map[string]interface{} `json:"telemetry,omitempty"`
}

func (c *StatusCollector) Snapshot(ctx context.Context) map[string]interface{} {
	payload := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if c.GatewayStatusProvider != nil {
		payload["gateway"] = c.GatewayStatusProvider()
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
	if c.ScheduleService != nil {
		items, err := c.ScheduleService.List(ctx, 200)
		if err != nil {
			payload["schedules"] = map[string]interface{}{"error": err.Error()}
		} else {
			payload["schedules"] = summarizeSchedules(items, time.Now().UTC())
		}
	}

	now := time.Now().UTC()
	usage := summarizeRuntimeUsage(
		c.OrchestratorLogBasePath,
		now,
		c.SessionWindow,
		c.ActiveSessionWindow,
		c.CostInputPer1K,
		c.CostOutputPer1K,
	)
	payload["sessions"] = usage.Sessions
	payload["subagents"] = usage.Subagent
	payload["cost"] = usage.Cost

	traceSummary := summarizeGatewayTrace(c.GatewayTraceBasePath, now, c.GatewayTraceWindow)
	payload["trace"] = map[string]interface{}{
		"window_minutes": traceSummary.WindowMinutes,
		"total_events":   traceSummary.TotalEvents,
		"error_events":   traceSummary.ErrorEvents,
		"by_event":       traceSummary.ByEvent,
		"error_by_event": traceSummary.ErrorByEvent,
		"by_channel":     traceSummary.ByChannel,
		"error_channels": traceSummary.ErrorChannels,
		"latest_at":      traceSummary.LatestAt,
	}

	executors := executor.ListExecutorCapabilities()
	payload["executors"] = executors
	payload["alerts"] = summarizeRuntimeAlerts(runtimeAlertInput{
		Now:                     now,
		Queue:                   c.Queue,
		Trace:                   traceSummary,
		Executors:               executors,
		RetryStormWindow:        c.AlertRetryStormWindow,
		RetryStormMinimum:       c.AlertRetryStormMinimum,
		QueueBacklogRatio:       c.AlertQueueBacklogRatio,
		QueueBacklogDepth:       c.AlertQueueBacklogDepth,
		ExecutorStrictAvailable: c.AlertExecutorStrictMode,
	})
	if c.ToolRuntime != nil {
		payload["tools"] = c.ToolRuntime.StatusSnapshot()
	}
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

type runtimeAlertInput struct {
	Now                     time.Time
	Queue                   *queue.Queue
	Trace                   gatewayTraceSummary
	Executors               []executor.ExecutorCapability
	RetryStormWindow        time.Duration
	RetryStormMinimum       int
	QueueBacklogRatio       float64
	QueueBacklogDepth       int
	ExecutorStrictAvailable bool
}

func summarizeRuntimeAlerts(input runtimeAlertInput) []runtimeAlert {
	now := input.Now.UTC()
	alerts := make([]runtimeAlert, 0)

	if input.Queue != nil {
		stats := input.Queue.Stats()
		depthThreshold := input.QueueBacklogDepth
		if depthThreshold <= 0 {
			depthThreshold = 8
		}
		ratioThreshold := input.QueueBacklogRatio
		if ratioThreshold <= 0 {
			ratioThreshold = 0.8
		}
		ratio := 0.0
		if stats.Capacity > 0 {
			ratio = float64(stats.Depth) / float64(stats.Capacity)
		}
		if stats.Started && stats.Depth >= depthThreshold && ratio >= ratioThreshold {
			alerts = append(alerts, runtimeAlert{
				Code:     "queue_backlog",
				Severity: "warning",
				Source:   "queue",
				Message:  fmt.Sprintf("queue depth=%d/%d exceeds backlog threshold", stats.Depth, stats.Capacity),
				Detected: now.Format(time.RFC3339),
				Telemetry: map[string]interface{}{
					"depth":    stats.Depth,
					"capacity": stats.Capacity,
					"ratio":    math.Round(ratio*1000) / 1000,
				},
			})
		}
	}

	retryStormMinimum := input.RetryStormMinimum
	if retryStormMinimum <= 0 {
		retryStormMinimum = 3
	}
	if input.Trace.ErrorByEvent["agent_process"] >= retryStormMinimum {
		alerts = append(alerts, runtimeAlert{
			Code:     "retry_storm",
			Severity: "critical",
			Source:   "gateway",
			Message:  fmt.Sprintf("agent_process errors reached %d in trace window", input.Trace.ErrorByEvent["agent_process"]),
			Detected: now.Format(time.RFC3339),
			Telemetry: map[string]interface{}{
				"event":          "agent_process",
				"error_count":    input.Trace.ErrorByEvent["agent_process"],
				"window_minutes": input.Trace.WindowMinutes,
			},
		})
	}

	if input.Trace.ErrorByEvent["channel_disconnected"] > 0 {
		alerts = append(alerts, runtimeAlert{
			Code:     "channel_disconnected",
			Severity: "critical",
			Source:   "gateway",
			Message:  "one or more channels exited unexpectedly",
			Detected: now.Format(time.RFC3339),
			Telemetry: map[string]interface{}{
				"events":         input.Trace.ErrorByEvent["channel_disconnected"],
				"error_channels": input.Trace.ErrorChannels,
			},
		})
	}

	if input.ExecutorStrictAvailable {
		missing := make([]string, 0)
		for _, entry := range input.Executors {
			if !entry.Installed {
				missing = append(missing, entry.Name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			alerts = append(alerts, runtimeAlert{
				Code:     "executor_unavailable",
				Severity: "warning",
				Source:   "executor",
				Message:  fmt.Sprintf("missing executor binaries: %s", strings.Join(missing, ", ")),
				Detected: now.Format(time.RFC3339),
				Telemetry: map[string]interface{}{
					"missing": missing,
				},
			})
		}
	}

	return alerts
}

func summarizeSchedules(items []schedulesvc.Job, now time.Time) map[string]interface{} {
	statusCounts := map[string]int{}
	kindCounts := map[string]int{}
	deliveryModeCounts := map[string]int{}
	nextRunAt := time.Time{}
	activeDueSoon := 0
	activeOverdue := 0

	for _, item := range items {
		statusCounts[item.Status]++
		kindCounts[item.Kind]++

		deliveryMode := strings.TrimSpace(item.Payload.Mode)
		if deliveryMode == "" {
			deliveryMode = schedulesvc.ModeDirect
		}
		deliveryModeCounts[deliveryMode]++

		if item.Status != schedulesvc.StatusActive || item.NextRunAt.IsZero() {
			continue
		}
		runAt := item.NextRunAt.UTC()
		if nextRunAt.IsZero() || runAt.Before(nextRunAt) {
			nextRunAt = runAt
		}
		if runAt.Before(now) {
			activeOverdue++
			continue
		}
		if !runAt.After(now.Add(5 * time.Minute)) {
			activeDueSoon++
		}
	}

	nextRunText := ""
	if !nextRunAt.IsZero() {
		nextRunText = nextRunAt.Format(time.RFC3339)
	}

	return map[string]interface{}{
		"total":              len(items),
		"status":             statusCounts,
		"kind":               kindCounts,
		"delivery_mode":      deliveryModeCounts,
		"active_due_in_5m":   activeDueSoon,
		"active_overdue":     activeOverdue,
		"next_active_run_at": nextRunText,
	}
}

func summarizeRuntimeUsage(basePath string, now time.Time, window time.Duration, activeWindow time.Duration, inputCostPer1K float64, outputCostPer1K float64) runtimeUsageSummary {
	if window <= 0 {
		window = 24 * time.Hour
	}
	if activeWindow <= 0 {
		activeWindow = 15 * time.Minute
	}
	since := now.Add(-window)
	entries := readOrchestratorLogsSince(basePath, since)

	sessionLastSeen := map[string]time.Time{}
	sessionChannel := map[string]string{}
	sessionUser := map[string]string{}
	subagentLastSeen := map[string]time.Time{}
	executorRuns := map[string]int{}

	promptChars := 0
	outputChars := 0
	errorRuns := 0

	for _, entry := range entries {
		sessionID := strings.TrimSpace(entry.SessionID)
		if sessionID == "" {
			sessionID = "unknown"
		}

		ts, err := parseRFC3339Any(entry.Timestamp)
		if err != nil {
			continue
		}
		ts = ts.UTC()
		if ts.Before(since) {
			continue
		}

		if seen, ok := sessionLastSeen[sessionID]; !ok || ts.After(seen) {
			sessionLastSeen[sessionID] = ts
			sessionChannel[sessionID] = strings.TrimSpace(entry.ChannelID)
			sessionUser[sessionID] = strings.TrimSpace(entry.UserID)
		}
		if isSubagentEntry(sessionID, entry.Stage) {
			if seen, ok := subagentLastSeen[sessionID]; !ok || ts.After(seen) {
				subagentLastSeen[sessionID] = ts
			}
		}

		executor := strings.TrimSpace(entry.Executor)
		if executor == "" {
			executor = "unknown"
		}
		executorRuns[executor]++

		if !strings.EqualFold(strings.TrimSpace(entry.Status), "ok") {
			errorRuns++
		}
		if entry.PromptChars > 0 {
			promptChars += entry.PromptChars
		}
		if entry.OutputChars > 0 {
			outputChars += entry.OutputChars
		}
	}

	activeSessions := 0
	activeSubagents := 0
	byChannel := map[string]int{}
	uniqueUsers := map[string]struct{}{}
	for sessionID, seenAt := range sessionLastSeen {
		if !seenAt.Before(now.Add(-activeWindow)) {
			activeSessions++
		}
		channel := strings.TrimSpace(sessionChannel[sessionID])
		if channel == "" {
			channel = "unknown"
		}
		byChannel[channel]++
		if userID := strings.TrimSpace(sessionUser[sessionID]); userID != "" {
			uniqueUsers[userID] = struct{}{}
		}
	}
	for _, seenAt := range subagentLastSeen {
		if !seenAt.Before(now.Add(-activeWindow)) {
			activeSubagents++
		}
	}

	inputTokens := estimateTokens(promptChars)
	outputTokens := estimateTokens(outputChars)
	totalTokens := inputTokens + outputTokens
	pricingConfigured := inputCostPer1K > 0 || outputCostPer1K > 0
	estimatedCost := ((float64(inputTokens) / 1000.0) * inputCostPer1K) + ((float64(outputTokens) / 1000.0) * outputCostPer1K)

	return runtimeUsageSummary{
		Sessions: map[string]interface{}{
			"window_hours":     int(window / time.Hour),
			"total":            len(sessionLastSeen),
			"active":           activeSessions,
			"active_window_ms": activeWindow.Milliseconds(),
			"unique_users":     len(uniqueUsers),
			"by_channel":       byChannel,
		},
		Subagent: map[string]interface{}{
			"window_hours":     int(window / time.Hour),
			"total_sessions":   len(subagentLastSeen),
			"active_sessions":  activeSubagents,
			"active_window_ms": activeWindow.Milliseconds(),
		},
		Cost: map[string]interface{}{
			"window_hours":            int(window / time.Hour),
			"executions":              len(entries),
			"error_executions":        errorRuns,
			"prompt_chars":            promptChars,
			"output_chars":            outputChars,
			"estimated_input_tokens":  inputTokens,
			"estimated_output_tokens": outputTokens,
			"estimated_total_tokens":  totalTokens,
			"estimated_cost_usd":      math.Round(estimatedCost*10000) / 10000,
			"pricing_configured":      pricingConfigured,
			"by_executor":             executorRuns,
		},
	}
}

func summarizeGatewayTrace(basePath string, now time.Time, window time.Duration) gatewayTraceSummary {
	if window <= 0 {
		window = 30 * time.Minute
	}
	since := now.Add(-window)
	entries := readGatewayTraceSince(basePath, since)
	summary := gatewayTraceSummary{
		WindowMinutes: int(window / time.Minute),
		ByEvent:       map[string]int{},
		ErrorByEvent:  map[string]int{},
		ByChannel:     map[string]int{},
		ErrorChannels: map[string]int{},
	}
	latest := time.Time{}
	for _, entry := range entries {
		timestamp, err := parseRFC3339Any(entry.Timestamp)
		if err != nil {
			continue
		}
		if timestamp.Before(since) {
			continue
		}
		summary.TotalEvents++
		event := strings.TrimSpace(entry.Event)
		if event == "" {
			event = "unknown"
		}
		summary.ByEvent[event]++

		channel := strings.TrimSpace(entry.ChannelID)
		if channel == "" {
			channel = "unknown"
		}
		summary.ByChannel[channel]++

		if strings.EqualFold(strings.TrimSpace(entry.Status), "error") {
			summary.ErrorEvents++
			summary.ErrorByEvent[event]++
			summary.ErrorChannels[channel]++
		}
		if latest.IsZero() || timestamp.After(latest) {
			latest = timestamp
		}
	}
	if !latest.IsZero() {
		summary.LatestAt = latest.UTC().Format(time.RFC3339)
	}
	return summary
}

func readGatewayTraceSince(basePath string, since time.Time) []gatewayTraceEntry {
	path := strings.TrimSpace(basePath)
	if path == "" {
		path = filepath.Join("output", "trace")
	}
	dirs, err := os.ReadDir(path)
	if err != nil {
		return []gatewayTraceEntry{}
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
	sort.Strings(dayDirs)

	entries := make([]gatewayTraceEntry, 0)
	for _, day := range dayDirs {
		dayAt, err := time.Parse("2006-01-02", day)
		if err != nil {
			continue
		}
		if dayAt.Add(24 * time.Hour).Before(since) {
			continue
		}
		items, err := readGatewayTraceFile(filepath.Join(path, day, "gateway_events.jsonl"), since)
		if err != nil {
			continue
		}
		entries = append(entries, items...)
	}
	return entries
}

func readGatewayTraceFile(path string, since time.Time) ([]gatewayTraceEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	out := make([]gatewayTraceEntry, 0, 64)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry gatewayTraceEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		ts, err := parseRFC3339Any(entry.Timestamp)
		if err != nil || ts.Before(since) {
			continue
		}
		out = append(out, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func estimateTokens(chars int) int {
	if chars <= 0 {
		return 0
	}
	return int(math.Ceil(float64(chars) / 4.0))
}

func isSubagentEntry(sessionID string, stage string) bool {
	lowerSession := strings.ToLower(strings.TrimSpace(sessionID))
	if strings.HasPrefix(lowerSession, "subagent-") || strings.HasPrefix(lowerSession, "subagent_") || strings.HasPrefix(lowerSession, "sa-") {
		return true
	}
	lowerStage := strings.ToLower(strings.TrimSpace(stage))
	return strings.Contains(lowerStage, "subagent")
}

func readOrchestratorLogsSince(basePath string, since time.Time) []orchestratorLogEntry {
	path := strings.TrimSpace(basePath)
	if path == "" {
		path = filepath.Join("output", "orchestrator")
	}
	dirs, err := os.ReadDir(path)
	if err != nil {
		return []orchestratorLogEntry{}
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
	sort.Strings(dayDirs)

	entries := make([]orchestratorLogEntry, 0)
	for _, day := range dayDirs {
		dayAt, err := time.Parse("2006-01-02", day)
		if err != nil {
			continue
		}
		if dayAt.Add(24 * time.Hour).Before(since) {
			continue
		}
		files, err := os.ReadDir(filepath.Join(path, day))
		if err != nil {
			continue
		}
		fileNames := make([]string, 0, len(files))
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := strings.TrimSpace(file.Name())
			if !strings.HasPrefix(name, "executor_") || !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			fileNames = append(fileNames, name)
		}
		sort.Strings(fileNames)

		for _, name := range fileNames {
			items, err := readOrchestratorLogFile(filepath.Join(path, day, name), since)
			if err != nil {
				continue
			}
			entries = append(entries, items...)
		}
	}
	return entries
}

func readOrchestratorLogFile(path string, since time.Time) ([]orchestratorLogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	out := make([]orchestratorLogEntry, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item orchestratorLogEntry
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		ts, err := parseRFC3339Any(item.Timestamp)
		if err != nil || ts.Before(since) {
			continue
		}
		out = append(out, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseRFC3339Any(value string) (time.Time, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Time{}, os.ErrInvalid
	}
	if ts, err := time.Parse(time.RFC3339Nano, text); err == nil {
		return ts, nil
	}
	return time.Parse(time.RFC3339, text)
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
