package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"alter0/app/core/agent"
)

func TestSnapshotIncludesExecutorCapabilities(t *testing.T) {
	collector := &StatusCollector{RepoPath: "."}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["executors"]
	if !ok {
		t.Fatalf("expected executors in snapshot")
	}

	executors, ok := raw.([]agent.ExecutorCapability)
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
