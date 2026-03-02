package channelchaos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildCandidatesFromTraceGeneratesFallbackScenario(t *testing.T) {
	now := time.Date(2026, time.March, 2, 1, 0, 0, 0, time.UTC)
	traceBase := t.TempDir()
	writeTraceDay(t, traceBase, now, []map[string]string{
		{"timestamp": now.Add(-4 * time.Minute).Format(time.RFC3339Nano), "channel_id": "slack", "event": "channel_disconnected", "status": "error", "detail": "socket closed"},
		{"timestamp": now.Add(-3 * time.Minute).Format(time.RFC3339Nano), "channel_id": "slack", "event": "inbound_received", "status": "ok"},
		{"timestamp": now.Add(-2 * time.Minute).Format(time.RFC3339Nano), "channel_id": "telegram", "event": "inbound_received", "status": "ok"},
	})

	report, err := BuildCandidatesFromTrace(traceBase, CandidateOptions{
		Now:                  now,
		Window:               10 * time.Minute,
		MinErrorEvents:       1,
		MaxCandidates:        3,
		MaxEventsPerScenario: 6,
	})
	if err != nil {
		t.Fatalf("BuildCandidatesFromTrace returned error: %v", err)
	}
	if report.CandidateCount != 1 {
		t.Fatalf("expected one candidate, got %d", report.CandidateCount)
	}

	candidate := report.Candidates[0]
	if !strings.HasPrefix(candidate.ID, "trace-sample-slack-") {
		t.Fatalf("unexpected candidate id: %q", candidate.ID)
	}
	if candidate.Expect.Status != "critical" {
		t.Fatalf("expected critical status, got %#v", candidate.Expect.Status)
	}
	if candidate.Expect.MinFallbackCandidates == nil || *candidate.Expect.MinFallbackCandidates != 1 {
		t.Fatalf("expected min fallback candidates=1, got %#v", candidate.Expect.MinFallbackCandidates)
	}
	if !contains(candidate.Expect.AlertCodes, "channel_disconnected") {
		t.Fatalf("expected channel_disconnected alert code, got %#v", candidate.Expect.AlertCodes)
	}

	fallbackFound := false
	for _, event := range candidate.Events {
		if event.ChannelID == "telegram" {
			fallbackFound = true
			break
		}
	}
	if !fallbackFound {
		t.Fatalf("expected fallback event from telegram, got %#v", candidate.Events)
	}
}

func TestBuildCandidatesFromTraceHonorsMinErrorsAndLimit(t *testing.T) {
	now := time.Date(2026, time.March, 2, 2, 0, 0, 0, time.UTC)
	traceBase := t.TempDir()
	writeTraceDay(t, traceBase, now, []map[string]string{
		{"timestamp": now.Add(-8 * time.Minute).Format(time.RFC3339Nano), "channel_id": "slack", "event": "agent_process", "status": "error", "detail": "temporary"},
		{"timestamp": now.Add(-7 * time.Minute).Format(time.RFC3339Nano), "channel_id": "slack", "event": "agent_process", "status": "error", "detail": "temporary"},
		{"timestamp": now.Add(-6 * time.Minute).Format(time.RFC3339Nano), "channel_id": "http", "event": "agent_process", "status": "error", "detail": "temporary"},
		{"timestamp": now.Add(-5 * time.Minute).Format(time.RFC3339Nano), "channel_id": "http", "event": "agent_process", "status": "error", "detail": "temporary"},
		{"timestamp": now.Add(-4 * time.Minute).Format(time.RFC3339Nano), "channel_id": "http", "event": "agent_process", "status": "error", "detail": "temporary"},
		{"timestamp": now.Add(-3 * time.Minute).Format(time.RFC3339Nano), "channel_id": "telegram", "event": "inbound_received", "status": "ok"},
	})

	report, err := BuildCandidatesFromTrace(traceBase, CandidateOptions{
		Now:            now,
		Window:         15 * time.Minute,
		MinErrorEvents: 3,
		MaxCandidates:  1,
	})
	if err != nil {
		t.Fatalf("BuildCandidatesFromTrace returned error: %v", err)
	}
	if report.CandidateCount != 1 {
		t.Fatalf("expected one candidate after limit, got %d", report.CandidateCount)
	}
	if !strings.Contains(report.Candidates[0].ID, "http") {
		t.Fatalf("expected top candidate to be http channel, got %q", report.Candidates[0].ID)
	}
}

func writeTraceDay(t *testing.T, traceBase string, now time.Time, events []map[string]string) {
	t.Helper()
	dayDir := filepath.Join(traceBase, now.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		t.Fatalf("mkdir trace day failed: %v", err)
	}

	lines := make([]string, 0, len(events))
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("marshal trace event failed: %v", err)
		}
		lines = append(lines, string(payload))
	}
	if err := os.WriteFile(filepath.Join(dayDir, "gateway_events.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("write trace file failed: %v", err)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
