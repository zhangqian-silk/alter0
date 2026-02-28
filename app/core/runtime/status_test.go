package runtime

import (
	"context"
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
