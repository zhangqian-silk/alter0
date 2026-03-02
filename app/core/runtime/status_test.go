package runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	executor "alter0/app/core/executor"
	"alter0/app/core/runtime/tools"
	schedulesvc "alter0/app/core/service/schedule"
	"alter0/app/pkg/queue"
	_ "modernc.org/sqlite"
)

func TestSnapshotIncludesExecutorCapabilities(t *testing.T) {
	collector := &StatusCollector{RepoPath: "."}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["executors"]
	if !ok {
		t.Fatalf("expected executors in snapshot")
	}

	executors, ok := raw.([]executor.ExecutorCapability)
	if !ok {
		t.Fatalf("expected executors to be []agent.ExecutorCapability, got %T", raw)
	}
	if len(executors) != 2 {
		t.Fatalf("expected 2 executor capabilities, got %d", len(executors))
	}

	for _, item := range executors {
		if item.Name == "" {
			t.Fatalf("executor entry missing name: %#v", item)
		}
		if item.Command == "" {
			t.Fatalf("executor entry missing command: %#v", item)
		}
	}
}

func TestSnapshotIncludesToolProtocolSnapshot(t *testing.T) {
	collector := &StatusCollector{
		RepoPath: ".",
		ToolRuntime: tools.NewRuntime(tools.Config{
			RequireConfirm: []string{"message"},
		}, t.TempDir()),
	}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["tools"]
	if !ok {
		t.Fatal("expected tools in snapshot")
	}
	payload, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected tools snapshot as map, got %T", raw)
	}
	protocol, ok := payload["protocol"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected protocol section, got %T", payload["protocol"])
	}
	if protocol["version"] != 1 {
		t.Fatalf("expected protocol version 1, got %#v", protocol["version"])
	}
	entries, ok := protocol["tools"].([]tools.Spec)
	if !ok {
		t.Fatalf("expected typed tool specs, got %T", protocol["tools"])
	}
	if len(entries) != 9 {
		t.Fatalf("expected 9 tool specs, got %d", len(entries))
	}
}

func TestSnapshotIncludesCommandAuditTail(t *testing.T) {
	baseDir := t.TempDir()
	dayDir := filepath.Join(baseDir, "2026-02-28")
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		t.Fatalf("failed to create audit dir: %v", err)
	}

	lines := "" +
		"{\"timestamp\":\"2026-02-28T18:00:00Z\",\"user_id\":\"u1\",\"channel_id\":\"cli\",\"request_id\":\"r1\",\"command\":\"help\",\"decision\":\"allow\"}\n" +
		"{\"timestamp\":\"2026-02-28T18:01:00Z\",\"user_id\":\"u1\",\"channel_id\":\"cli\",\"request_id\":\"r2\",\"command\":\"status\",\"decision\":\"allow\"}\n" +
		"{\"timestamp\":\"2026-02-28T18:02:00Z\",\"user_id\":\"u2\",\"channel_id\":\"http\",\"request_id\":\"r3\",\"command\":\"executor\",\"decision\":\"deny\",\"reason\":\"permission denied\"}\n"
	if err := os.WriteFile(filepath.Join(dayDir, "command_permission.jsonl"), []byte(lines), 0644); err != nil {
		t.Fatalf("failed to seed audit log: %v", err)
	}

	collector := &StatusCollector{
		RepoPath:             ".",
		CommandAuditBasePath: baseDir,
		CommandAuditTailSize: 2,
	}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["command_audit_tail"]
	if !ok {
		t.Fatalf("expected command_audit_tail in snapshot")
	}

	tail, ok := raw.([]commandAuditTailEntry)
	if !ok {
		t.Fatalf("expected command_audit_tail to be []commandAuditTailEntry, got %T", raw)
	}
	if len(tail) != 2 {
		t.Fatalf("expected 2 audit tail entries, got %d", len(tail))
	}
	if tail[0].RequestID != "r2" || tail[1].RequestID != "r3" {
		t.Fatalf("expected newest 2 entries in chronological order, got %#v", tail)
	}
	if tail[1].Reason != "permission denied" {
		t.Fatalf("expected reason in latest entry, got %#v", tail[1])
	}
}

func TestSnapshotIncludesSessionSubagentAndCostMetrics(t *testing.T) {
	now := time.Now().UTC()
	baseDir := t.TempDir()
	dayDir := filepath.Join(baseDir, now.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		t.Fatalf("failed to create orchestrator log dir: %v", err)
	}

	entries := []map[string]interface{}{
		{
			"timestamp":    now.Add(-10 * time.Minute).Format(time.RFC3339Nano),
			"session_id":   "session-main-1",
			"stage":        "gen",
			"executor":     "codex",
			"status":       "ok",
			"prompt_chars": 400,
			"output_chars": 120,
			"user_id":      "u1",
			"channel_id":   "cli",
		},
		{
			"timestamp":    now.Add(-5 * time.Minute).Format(time.RFC3339Nano),
			"session_id":   "subagent-42",
			"stage":        "subagent-gen",
			"executor":     "codex",
			"status":       "error",
			"prompt_chars": 800,
			"output_chars": 40,
			"user_id":      "u1",
			"channel_id":   "cli",
		},
		{
			"timestamp":    now.Add(-2 * time.Hour).Format(time.RFC3339Nano),
			"session_id":   "session-main-2",
			"stage":        "gen",
			"executor":     "claude_code",
			"status":       "ok",
			"prompt_chars": 160,
			"output_chars": 200,
			"user_id":      "u2",
			"channel_id":   "http",
		},
	}

	logPath := filepath.Join(dayDir, fmt.Sprintf("executor_%s.jsonl", now.Format("20060102-15")))
	var lines []string
	for _, item := range entries {
		payload, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("marshal log entry failed: %v", err)
		}
		lines = append(lines, string(payload))
	}
	if err := os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("write orchestrator log failed: %v", err)
	}

	collector := &StatusCollector{
		RepoPath:                ".",
		OrchestratorLogBasePath: baseDir,
		SessionWindow:           24 * time.Hour,
		ActiveSessionWindow:     30 * time.Minute,
		CostInputPer1K:          0.002,
		CostOutputPer1K:         0.006,
	}

	snap := collector.Snapshot(context.Background())

	sessions, ok := snap["sessions"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sessions map, got %T", snap["sessions"])
	}
	if got := intValue(t, sessions["total"]); got != 3 {
		t.Fatalf("expected sessions total=3, got %d", got)
	}
	if got := intValue(t, sessions["active"]); got != 2 {
		t.Fatalf("expected active sessions=2, got %d", got)
	}

	byChannel, ok := sessions["by_channel"].(map[string]int)
	if !ok {
		t.Fatalf("expected by_channel map[string]int, got %T", sessions["by_channel"])
	}
	if byChannel["cli"] != 2 || byChannel["http"] != 1 {
		t.Fatalf("unexpected by_channel stats: %#v", byChannel)
	}

	subagents, ok := snap["subagents"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected subagents map, got %T", snap["subagents"])
	}
	if got := intValue(t, subagents["total_sessions"]); got != 1 {
		t.Fatalf("expected subagent sessions=1, got %d", got)
	}
	if got := intValue(t, subagents["active_sessions"]); got != 1 {
		t.Fatalf("expected active subagent sessions=1, got %d", got)
	}

	cost, ok := snap["cost"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cost map, got %T", snap["cost"])
	}
	if got := intValue(t, cost["executions"]); got != 3 {
		t.Fatalf("expected executions=3, got %d", got)
	}
	if got := intValue(t, cost["error_executions"]); got != 1 {
		t.Fatalf("expected error_executions=1, got %d", got)
	}
	if got := intValue(t, cost["estimated_total_tokens"]); got != 430 {
		t.Fatalf("expected estimated_total_tokens=430, got %d", got)
	}
	if got, ok := cost["pricing_configured"].(bool); !ok || !got {
		t.Fatalf("expected pricing_configured=true, got %#v", cost["pricing_configured"])
	}
	hotspots, ok := cost["session_hotspots"].([]runtimeSessionCostHotspot)
	if !ok {
		t.Fatalf("expected session_hotspots as []runtimeSessionCostHotspot, got %T", cost["session_hotspots"])
	}
	if len(hotspots) != 3 {
		t.Fatalf("expected 3 hotspots, got %#v", hotspots)
	}
	if hotspots[0].SessionID != "subagent-42" {
		t.Fatalf("expected top hotspot to be subagent-42, got %#v", hotspots[0])
	}
}

func TestSnapshotIncludesExpandedScheduleMetrics(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:runtime-status-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer db.Close()

	store, err := schedulesvc.NewStore(db)
	if err != nil {
		t.Fatalf("create schedule store failed: %v", err)
	}

	now := time.Now().UTC()
	seed := []schedulesvc.Job{
		{
			ID:        "at-active-overdue",
			Kind:      schedulesvc.KindAt,
			Spec:      now.Add(-1 * time.Minute).Format(time.RFC3339),
			Status:    schedulesvc.StatusActive,
			Payload:   schedulesvc.DeliveryPayload{Mode: schedulesvc.ModeDirect, ChannelID: "cli", To: "u1", Content: "ping"},
			NextRunAt: now.Add(-1 * time.Minute),
		},
		{
			ID:        "cron-active-soon",
			Kind:      schedulesvc.KindCron,
			Spec:      "*/5 * * * *",
			Status:    schedulesvc.StatusActive,
			Payload:   schedulesvc.DeliveryPayload{Mode: schedulesvc.ModeAgentTurn, ChannelID: "cli", To: "u1", Content: "ping", AgentID: "default"},
			NextRunAt: now.Add(3 * time.Minute),
		},
		{
			ID:      "paused",
			Kind:    schedulesvc.KindCron,
			Spec:    "*/10 * * * *",
			Status:  schedulesvc.StatusPaused,
			Payload: schedulesvc.DeliveryPayload{Mode: schedulesvc.ModeDirect, ChannelID: "cli", To: "u1", Content: "paused"},
		},
	}
	for _, item := range seed {
		if _, err := store.Create(context.Background(), item); err != nil {
			t.Fatalf("seed schedule failed: %v", err)
		}
	}

	service := schedulesvc.NewService(store, nil)
	collector := &StatusCollector{RepoPath: ".", ScheduleService: service}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["schedules"]
	if !ok {
		t.Fatal("expected schedules section in snapshot")
	}
	stats, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected schedules map, got %T", raw)
	}
	if got := intValue(t, stats["total"]); got != 3 {
		t.Fatalf("expected total schedules=3, got %d", got)
	}
	if got := intValue(t, stats["active_overdue"]); got != 1 {
		t.Fatalf("expected active_overdue=1, got %d", got)
	}
	if got := intValue(t, stats["active_due_in_5m"]); got != 1 {
		t.Fatalf("expected active_due_in_5m=1, got %d", got)
	}

	kind, ok := stats["kind"].(map[string]int)
	if !ok {
		t.Fatalf("expected kind map[string]int, got %T", stats["kind"])
	}
	if kind[schedulesvc.KindAt] != 1 || kind[schedulesvc.KindCron] != 2 {
		t.Fatalf("unexpected kind metrics: %#v", kind)
	}

	deliveryMode, ok := stats["delivery_mode"].(map[string]int)
	if !ok {
		t.Fatalf("expected delivery_mode map[string]int, got %T", stats["delivery_mode"])
	}
	if deliveryMode[schedulesvc.ModeDirect] != 2 || deliveryMode[schedulesvc.ModeAgentTurn] != 1 {
		t.Fatalf("unexpected delivery_mode metrics: %#v", deliveryMode)
	}

	nextRunAt, ok := stats["next_active_run_at"].(string)
	if !ok || strings.TrimSpace(nextRunAt) == "" {
		t.Fatalf("expected next_active_run_at to be non-empty string, got %#v", stats["next_active_run_at"])
	}
}

func TestSnapshotIncludesQueueInFlight(t *testing.T) {
	q := queue.New(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start queue failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	if _, err := q.Enqueue(queue.Job{Run: func(context.Context) error {
		started <- struct{}{}
		<-release
		return nil
	}}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected queue job to start")
	}

	collector := &StatusCollector{Queue: q}
	snap := collector.Snapshot(context.Background())
	raw, ok := snap["queue"]
	if !ok {
		t.Fatal("expected queue stats in snapshot")
	}

	stats, ok := raw.(queue.Stats)
	if !ok {
		t.Fatalf("expected queue stats type, got %T", raw)
	}
	if stats.Workers != 1 {
		t.Fatalf("expected workers=1, got %+v", stats)
	}
	if stats.InFlight != 1 {
		t.Fatalf("expected in_flight=1, got %+v", stats)
	}

	close(release)
}

func TestSnapshotIncludesQueueLastShutdownReport(t *testing.T) {
	q := queue.New(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start queue failed: %v", err)
	}
	if _, err := q.Enqueue(queue.Job{Run: func(context.Context) error { return nil }}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if _, err := q.StopWithReport(200 * time.Millisecond); err != nil {
		t.Fatalf("stop queue failed: %v", err)
	}

	collector := &StatusCollector{Queue: q}
	snap := collector.Snapshot(context.Background())
	raw, ok := snap["queue"]
	if !ok {
		t.Fatal("expected queue stats in snapshot")
	}

	stats, ok := raw.(queue.Stats)
	if !ok {
		t.Fatalf("expected queue stats type, got %T", raw)
	}
	if stats.LastShutdown == nil {
		t.Fatalf("expected last_shutdown report in queue stats, got %+v", stats)
	}
	if stats.LastShutdown.TimedOut {
		t.Fatalf("expected graceful shutdown report, got %+v", stats.LastShutdown)
	}
}

func TestSnapshotIncludesAgentRegistry(t *testing.T) {
	collector := &StatusCollector{
		AgentEntries: []AgentEntry{
			{AgentID: "beta", Workspace: "/tmp/w2", AgentDir: "/tmp/a2", Executor: "codex"},
			{AgentID: "alpha", Workspace: "/tmp/w1", AgentDir: "/tmp/a1", Executor: "claude_code"},
		},
	}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["agents"]
	if !ok {
		t.Fatal("expected agents in snapshot")
	}
	items, ok := raw.([]AgentEntry)
	if !ok {
		t.Fatalf("expected []AgentEntry, got %T", raw)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(items))
	}
	if items[0].AgentID != "alpha" || items[1].AgentID != "beta" {
		t.Fatalf("expected sorted agents [alpha beta], got %+v", items)
	}
}

func TestSnapshotIncludesRiskWatchlistSummaryAndAlerts(t *testing.T) {
	now := time.Now().UTC()
	watchlistPath := filepath.Join(t.TempDir(), "risk-watchlist.json")
	doc := riskWatchlistDocument{
		UpdatedAt: now.Add(-96 * time.Hour).Format(time.RFC3339),
		Items: []riskWatchlistItem{
			{
				ID:           "provider-policy-drift",
				Category:     "provider_policy",
				Severity:     "high",
				Status:       "watch",
				NextReviewAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
				Owner:        "runtime",
			},
			{
				ID:           "skill-provenance",
				Category:     "supply_chain",
				Severity:     "medium",
				Status:       "mitigated",
				NextReviewAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
				Owner:        "runtime",
			},
		},
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal watchlist failed: %v", err)
	}
	if err := os.WriteFile(watchlistPath, payload, 0644); err != nil {
		t.Fatalf("write watchlist failed: %v", err)
	}

	collector := &StatusCollector{
		RiskWatchlistPath:       watchlistPath,
		RiskWatchlistStaleAfter: 72 * time.Hour,
	}
	snap := collector.Snapshot(context.Background())

	risk, ok := snap["risk_watchlist"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected risk_watchlist map, got %T", snap["risk_watchlist"])
	}
	if risk["status"] != "ok" {
		t.Fatalf("expected risk watchlist status ok, got %#v", risk["status"])
	}
	if got := intValue(t, risk["overdue_items"]); got != 1 {
		t.Fatalf("expected overdue_items=1, got %d", got)
	}

	alerts, ok := snap["alerts"].([]runtimeAlert)
	if !ok {
		t.Fatalf("expected alerts list, got %T", snap["alerts"])
	}
	codes := map[string]bool{}
	criticalOverdue := false
	for _, alert := range alerts {
		codes[alert.Code] = true
		if alert.Code == "risk_watchlist_item_overdue" && alert.Severity == "critical" {
			criticalOverdue = true
		}
	}
	if !codes["risk_watchlist_stale"] {
		t.Fatalf("expected risk_watchlist_stale in alerts: %+v", alerts)
	}
	if !codes["risk_watchlist_item_overdue"] {
		t.Fatalf("expected risk_watchlist_item_overdue in alerts: %+v", alerts)
	}
	if !criticalOverdue {
		t.Fatalf("expected critical overdue alert, got %+v", alerts)
	}
}

func TestSnapshotIncludesRiskWatchlistMissingAlert(t *testing.T) {
	collector := &StatusCollector{
		RiskWatchlistPath: filepath.Join(t.TempDir(), "missing.json"),
	}
	snap := collector.Snapshot(context.Background())

	risk, ok := snap["risk_watchlist"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected risk_watchlist map, got %T", snap["risk_watchlist"])
	}
	if risk["status"] != "missing" {
		t.Fatalf("expected missing risk watchlist status, got %#v", risk["status"])
	}

	alerts, ok := snap["alerts"].([]runtimeAlert)
	if !ok {
		t.Fatalf("expected alerts list, got %T", snap["alerts"])
	}
	missingAlert := false
	for _, alert := range alerts {
		if alert.Code == "risk_watchlist_missing" {
			missingAlert = true
			break
		}
	}
	if !missingAlert {
		t.Fatalf("expected risk_watchlist_missing alert, got %+v", alerts)
	}
}

func TestSnapshotIncludesTraceSummaryAndAlerts(t *testing.T) {
	now := time.Now().UTC()
	traceBase := t.TempDir()
	dayDir := filepath.Join(traceBase, now.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		t.Fatalf("create trace dir failed: %v", err)
	}
	entries := []string{
		fmt.Sprintf(`{"timestamp":"%s","channel_id":"cli","event":"inbound_received","status":"ok"}`, now.Add(-3*time.Minute).Format(time.RFC3339Nano)),
		fmt.Sprintf(`{"timestamp":"%s","channel_id":"cli","event":"agent_process","status":"error","detail":"temporary"}`, now.Add(-2*time.Minute).Format(time.RFC3339Nano)),
		fmt.Sprintf(`{"timestamp":"%s","channel_id":"cli","event":"agent_process","status":"error","detail":"temporary"}`, now.Add(-90*time.Second).Format(time.RFC3339Nano)),
		fmt.Sprintf(`{"timestamp":"%s","channel_id":"telegram","event":"agent_process","status":"error","detail":"temporary"}`, now.Add(-45*time.Second).Format(time.RFC3339Nano)),
		fmt.Sprintf(`{"timestamp":"%s","channel_id":"slack","event":"channel_disconnected","status":"error","detail":"connection dropped"}`, now.Add(-30*time.Second).Format(time.RFC3339Nano)),
	}
	if err := os.WriteFile(filepath.Join(dayDir, "gateway_events.jsonl"), []byte(strings.Join(entries, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("write trace file failed: %v", err)
	}

	q := queue.New(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("queue start failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)
	for i := 0; i < 8; i++ {
		if _, err := q.Enqueue(queue.Job{Run: func(context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}}); err != nil {
			t.Fatalf("enqueue job failed: %v", err)
		}
	}
	time.Sleep(20 * time.Millisecond)

	collector := &StatusCollector{
		Queue:                   q,
		GatewayTraceBasePath:    traceBase,
		GatewayTraceWindow:      10 * time.Minute,
		AlertRetryStormMinimum:  3,
		AlertQueueBacklogRatio:  0.7,
		AlertQueueBacklogDepth:  6,
		AlertExecutorStrictMode: true,
	}

	snap := collector.Snapshot(context.Background())
	trace, ok := snap["trace"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected trace summary map, got %T", snap["trace"])
	}
	if got := intValue(t, trace["error_events"]); got != 4 {
		t.Fatalf("expected trace error events=4, got %d", got)
	}
	disconnectedByChannel, ok := trace["disconnected_by_channel"].(map[string]int)
	if !ok {
		t.Fatalf("expected disconnected_by_channel map[string]int, got %T", trace["disconnected_by_channel"])
	}
	if disconnectedByChannel["slack"] != 1 {
		t.Fatalf("expected slack disconnected count=1, got %#v", disconnectedByChannel)
	}

	channelDegradation, ok := snap["channel_degradation"].(channelDegradationSummary)
	if !ok {
		t.Fatalf("expected channel_degradation summary, got %T", snap["channel_degradation"])
	}
	if channelDegradation.Status != "critical" {
		t.Fatalf("expected channel_degradation status critical, got %#v", channelDegradation.Status)
	}
	if channelDegradation.DegradedChannels == 0 {
		t.Fatalf("expected degraded channels > 0, got %#v", channelDegradation)
	}

	alerts, ok := snap["alerts"].([]runtimeAlert)
	if !ok {
		t.Fatalf("expected typed runtime alerts, got %T", snap["alerts"])
	}
	if len(alerts) < 4 {
		t.Fatalf("expected at least 4 alerts, got %+v", alerts)
	}
	codes := map[string]bool{}
	for _, alert := range alerts {
		codes[alert.Code] = true
	}
	for _, code := range []string{"queue_backlog", "retry_storm", "channel_disconnected", "channel_degradation"} {
		if !codes[code] {
			t.Fatalf("expected alert %q in %+v", code, alerts)
		}
	}
}

func TestSummarizeChannelDegradationFallbackCandidates(t *testing.T) {
	summary := summarizeChannelDegradation(gatewayTraceSummary{
		ByChannel: map[string]int{
			"slack":    4,
			"telegram": 5,
		},
		ErrorChannels: map[string]int{
			"slack": 2,
		},
		DisconnectedByChannel: map[string]int{},
	})

	if summary.Status != "degraded" {
		t.Fatalf("expected degraded status, got %#v", summary.Status)
	}
	if summary.HealthyChannels != 1 {
		t.Fatalf("expected one healthy channel, got %#v", summary.HealthyChannels)
	}
	if len(summary.FallbackCandidates) != 1 || summary.FallbackCandidates[0] != "telegram" {
		t.Fatalf("expected telegram fallback candidate, got %#v", summary.FallbackCandidates)
	}
	if len(summary.Channels) != 1 || summary.Channels[0].ChannelID != "slack" {
		t.Fatalf("expected slack degradation entry, got %#v", summary.Channels)
	}
}

func TestSummarizeChannelDegradationWithThresholdOverrides(t *testing.T) {
	summary := summarizeChannelDegradationWithThresholds(
		gatewayTraceSummary{
			ByChannel: map[string]int{
				"cli":      6,
				"slack":    3,
				"telegram": 5,
			},
			ErrorChannels: map[string]int{
				"slack":    1,
				"telegram": 3,
			},
			DisconnectedByChannel: map[string]int{},
		},
		ChannelDegradationThresholds{
			MinEvents:                     1,
			WarningErrorRateThreshold:     0.001,
			CriticalErrorRateThreshold:    0.5,
			CriticalErrorCountThreshold:   3,
			CriticalDisconnectedThreshold: 1,
		},
		map[string]ChannelDegradationThresholds{
			"slack": {
				MinEvents:                     5,
				WarningErrorRateThreshold:     0.3,
				CriticalErrorRateThreshold:    0.7,
				CriticalErrorCountThreshold:   4,
				CriticalDisconnectedThreshold: 1,
			},
		},
	)

	if summary.Status != "critical" {
		t.Fatalf("expected critical status, got %#v", summary.Status)
	}
	if summary.DegradedChannels != 1 {
		t.Fatalf("expected one degraded channel, got %#v", summary.DegradedChannels)
	}
	if summary.CriticalChannels != 1 {
		t.Fatalf("expected one critical channel, got %#v", summary.CriticalChannels)
	}
	if summary.SuppressedChannels != 1 {
		t.Fatalf("expected one suppressed channel, got %#v", summary.SuppressedChannels)
	}
	if len(summary.FallbackCandidates) != 1 || summary.FallbackCandidates[0] != "cli" {
		t.Fatalf("expected cli fallback candidate, got %#v", summary.FallbackCandidates)
	}

	var slackEntry channelDegradationEntry
	var telegramEntry channelDegradationEntry
	for _, entry := range summary.Channels {
		switch entry.ChannelID {
		case "slack":
			slackEntry = entry
		case "telegram":
			telegramEntry = entry
		}
	}

	if slackEntry.Severity != "suppressed" {
		t.Fatalf("expected slack severity suppressed, got %#v", slackEntry)
	}
	if slackEntry.ThresholdProfile != "channel:slack" {
		t.Fatalf("expected slack threshold profile override, got %#v", slackEntry.ThresholdProfile)
	}
	if telegramEntry.Severity != "critical" {
		t.Fatalf("expected telegram severity critical, got %#v", telegramEntry)
	}
}

func TestSnapshotIncludesSessionCostPressureAlerts(t *testing.T) {
	now := time.Now().UTC()
	baseDir := t.TempDir()
	dayDir := filepath.Join(baseDir, now.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		t.Fatalf("failed to create orchestrator log dir: %v", err)
	}

	entries := []map[string]interface{}{
		{
			"timestamp":    now.Add(-3 * time.Minute).Format(time.RFC3339Nano),
			"session_id":   "session-heavy",
			"stage":        "gen",
			"executor":     "codex",
			"status":       "ok",
			"prompt_chars": 12000,
			"output_chars": 160,
			"user_id":      "u1",
			"channel_id":   "cli",
		},
		{
			"timestamp":    now.Add(-2 * time.Minute).Format(time.RFC3339Nano),
			"session_id":   "session-light",
			"stage":        "gen",
			"executor":     "claude_code",
			"status":       "ok",
			"prompt_chars": 500,
			"output_chars": 800,
			"user_id":      "u2",
			"channel_id":   "http",
		},
	}

	logPath := filepath.Join(dayDir, fmt.Sprintf("executor_%s.jsonl", now.Format("20060102-15")))
	var lines []string
	for _, item := range entries {
		payload, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("marshal log entry failed: %v", err)
		}
		lines = append(lines, string(payload))
	}
	if err := os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("write orchestrator log failed: %v", err)
	}

	collector := &StatusCollector{
		OrchestratorLogBasePath:    baseDir,
		SessionWindow:              24 * time.Hour,
		ActiveSessionWindow:        30 * time.Minute,
		AlertSessionCostShare:      0.4,
		AlertSessionCostMinTokens:  1000,
		AlertSessionPromptOutRatio: 8,
	}
	snap := collector.Snapshot(context.Background())

	alerts, ok := snap["alerts"].([]runtimeAlert)
	if !ok {
		t.Fatalf("expected typed runtime alerts, got %T", snap["alerts"])
	}
	codes := map[string]bool{}
	for _, alert := range alerts {
		codes[alert.Code] = true
	}
	for _, code := range []string{"session_cost_hotspot", "session_compaction_pressure"} {
		if !codes[code] {
			t.Fatalf("expected alert %q in %+v", code, alerts)
		}
	}

	cost, ok := snap["cost"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cost map, got %T", snap["cost"])
	}
	hotspots, ok := cost["session_hotspots"].([]runtimeSessionCostHotspot)
	if !ok || len(hotspots) == 0 {
		t.Fatalf("expected session hotspots in cost map, got %#v", cost["session_hotspots"])
	}
	if hotspots[0].SessionID != "session-heavy" {
		t.Fatalf("expected session-heavy hotspot first, got %#v", hotspots[0])
	}
	if hotspots[0].Share < 0.8 {
		t.Fatalf("expected heavy hotspot share >= 0.8, got %#v", hotspots[0])
	}

	guidance, ok := cost["threshold_guidance"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected threshold guidance map, got %T", cost["threshold_guidance"])
	}
	if guidance["status"] != "ok" {
		t.Fatalf("expected threshold guidance status ok, got %#v", guidance["status"])
	}
	if sample := intValue(t, guidance["sample_sessions"]); sample != 1 {
		t.Fatalf("expected one heavy session sample, got %d", sample)
	}
	recommended, ok := guidance["recommended"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected recommended threshold map, got %T", guidance["recommended"])
	}
	if share, ok := recommended["session_cost_share"].(float64); !ok || share < 0.8 {
		t.Fatalf("expected recommended share >= 0.8, got %#v", recommended["session_cost_share"])
	}

	workloadTiers, ok := guidance["workload_tiers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected workload_tiers map, got %T", guidance["workload_tiers"])
	}
	if workloadTiers["status"] != "ok" {
		t.Fatalf("expected workload_tiers status ok, got %#v", workloadTiers["status"])
	}
	tiers, ok := workloadTiers["tiers"].([]runtimeWorkloadTierGuidance)
	if !ok || len(tiers) == 0 {
		t.Fatalf("expected workload tier guidance entries, got %#v", workloadTiers["tiers"])
	}
	if tiers[0].Name != "2x_to_4x_min_tokens" {
		t.Fatalf("expected heavy session in 2x_to_4x_min_tokens, got %#v", tiers[0])
	}
	if tiers[0].SampleSessions != 1 {
		t.Fatalf("expected one tier sample, got %#v", tiers[0].SampleSessions)
	}
}

func TestSnapshotSessionCostThresholdGuidanceInsufficientData(t *testing.T) {
	collector := &StatusCollector{}
	snap := collector.Snapshot(context.Background())

	cost, ok := snap["cost"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cost map, got %T", snap["cost"])
	}
	guidance, ok := cost["threshold_guidance"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected threshold guidance map, got %T", cost["threshold_guidance"])
	}
	if guidance["status"] != "insufficient_data" {
		t.Fatalf("expected insufficient_data guidance, got %#v", guidance["status"])
	}
	workloadTiers, ok := guidance["workload_tiers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected workload_tiers map, got %T", guidance["workload_tiers"])
	}
	if workloadTiers["status"] != "insufficient_data" {
		t.Fatalf("expected workload_tiers insufficient_data, got %#v", workloadTiers["status"])
	}
}

func TestSnapshotIncludesGitDivergenceAgainstUpstream(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	runGitCmd(t, base, "git", "init", "--bare", remote)

	seed := filepath.Join(base, "seed")
	runGitCmd(t, base, "git", "clone", remote, seed)
	writeRepoFile(t, seed, "README.md", "seed\n")
	runGitCmd(t, seed, "git", "add", "README.md")
	runGitCmd(t, seed, "git", "commit", "-m", "seed")
	runGitCmd(t, seed, "git", "push", "origin", "master")

	local := filepath.Join(base, "local")
	runGitCmd(t, base, "git", "clone", remote, local)
	writeRepoFile(t, local, "local.txt", "local\n")
	runGitCmd(t, local, "git", "add", "local.txt")
	runGitCmd(t, local, "git", "commit", "-m", "local commit")

	other := filepath.Join(base, "other")
	runGitCmd(t, base, "git", "clone", remote, other)
	writeRepoFile(t, other, "remote.txt", "remote\n")
	runGitCmd(t, other, "git", "add", "remote.txt")
	runGitCmd(t, other, "git", "commit", "-m", "remote commit")
	runGitCmd(t, other, "git", "push", "origin", "master")

	runGitCmd(t, local, "git", "fetch", "origin")

	collector := &StatusCollector{RepoPath: local}
	snap := collector.Snapshot(context.Background())
	raw, ok := snap["git"]
	if !ok {
		t.Fatal("expected git status in snapshot")
	}

	git, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected git status map, got %T", raw)
	}
	if git["upstream"] != "origin/master" {
		t.Fatalf("expected upstream origin/master, got %#v", git["upstream"])
	}
	if git["ahead"] != 1 {
		t.Fatalf("expected ahead=1, got %#v", git["ahead"])
	}
	if git["behind"] != 1 {
		t.Fatalf("expected behind=1, got %#v", git["behind"])
	}
}

func runGitCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=alter0-test",
		"GIT_AUTHOR_EMAIL=alter0-test@example.com",
		"GIT_COMMITTER_NAME=alter0-test",
		"GIT_COMMITTER_EMAIL=alter0-test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed (%s %s): %v\n%s", name, strings.Join(args, " "), err, string(out))
	}
}

func writeRepoFile(t *testing.T, repoPath, relPath, content string) {
	t.Helper()
	absPath := filepath.Join(repoPath, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func intValue(t *testing.T, v interface{}) int {
	t.Helper()
	switch value := v.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		t.Fatalf("expected numeric value, got %T", v)
		return 0
	}
}
