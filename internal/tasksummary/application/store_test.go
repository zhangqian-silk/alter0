package application

import (
	"testing"
	"time"

	taskdomain "alter0/internal/task/domain"
)

func TestStoreSnapshotInjectsRecentWindowByDefault(t *testing.T) {
	store := NewStore(Options{RecentWindow: 5, DeepTopK: 5})
	now := time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)

	for idx := 0; idx < 8; idx++ {
		task := buildTerminalTask(
			"task-default-"+time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC).Add(time.Duration(idx)*time.Minute).Format("150405"),
			"session-default",
			taskdomain.TaskStatusSuccess,
			"release",
			"finish release step",
			now.Add(time.Duration(idx)*time.Minute),
		)
		store.Record(task)
	}

	snapshot := store.Snapshot("帮我延续当前任务", now.Add(10*time.Minute))
	if snapshot.Mode != RetrievalModeRecent {
		t.Fatalf("expected recent mode, got %q", snapshot.Mode)
	}
	if snapshot.DeepTriggered {
		t.Fatalf("expected deep retrieval not triggered")
	}
	if len(snapshot.Summaries) != 5 {
		t.Fatalf("expected 5 summaries by default window, got %d", len(snapshot.Summaries))
	}
	if snapshot.Summaries[0].TaskID != "task-default-090700" {
		t.Fatalf("expected latest summary first, got %q", snapshot.Summaries[0].TaskID)
	}
}

func TestStoreSnapshotTriggersDeepRetrievalAndDrillDown(t *testing.T) {
	store := NewStore(Options{RecentWindow: 5, DeepTopK: 5, DetailLogLimit: 2, DetailArtifactLimit: 1})
	now := time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)

	store.Record(buildTerminalTask("task-recent", "session-a", taskdomain.TaskStatusSuccess, "release", "recent deploy result", now.Add(-20*time.Minute)))
	legacy := buildTerminalTask("task-legacy", "session-b", taskdomain.TaskStatusFailed, "migration", "legacy migration failed", now.Add(-7*24*time.Hour))
	legacy.Logs = append(legacy.Logs,
		taskdomain.TaskLog{Timestamp: now.Add(-7*24*time.Hour + time.Minute), Level: taskdomain.TaskLogLevelError, Message: "db timeout"},
		taskdomain.TaskLog{Timestamp: now.Add(-7*24*time.Hour + 2*time.Minute), Level: taskdomain.TaskLogLevelWarn, Message: "retry exhausted"},
	)
	legacy.Artifacts = append(legacy.Artifacts,
		taskdomain.TaskArtifact{ArtifactID: "a1", Name: "error.log", ContentType: "text/plain", Content: "stacktrace", CreatedAt: now.Add(-7*24*time.Hour + 3*time.Minute)},
	)
	store.Record(legacy)

	query := "回顾上周那个 migration 任务，给我 task-legacy 的日志细节"
	snapshot := store.Snapshot(query, now)
	if !snapshot.DeepTriggered {
		t.Fatalf("expected deep retrieval triggered")
	}
	if snapshot.Mode != RetrievalModeDeep {
		t.Fatalf("expected deep mode, got %q", snapshot.Mode)
	}
	if len(snapshot.Summaries) == 0 || snapshot.Summaries[0].TaskID != "task-legacy" {
		t.Fatalf("expected deep retrieval hit task-legacy, got %+v", snapshot.Summaries)
	}
	if len(snapshot.Details) != 1 {
		t.Fatalf("expected detail drill-down for explicit detail request, got %d", len(snapshot.Details))
	}
	if snapshot.Details[0].TaskID != "task-legacy" {
		t.Fatalf("expected detail task_id task-legacy, got %q", snapshot.Details[0].TaskID)
	}
	if len(snapshot.Details[0].Logs) == 0 || len(snapshot.Details[0].Artifacts) == 0 {
		t.Fatalf("expected logs and artifacts in detail drill-down")
	}
}

func TestStoreSnapshotPreventsFalseTriggerAndFallbackOnMiss(t *testing.T) {
	store := NewStore(Options{RecentWindow: 4, DeepTopK: 3})
	now := time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)

	store.Record(buildTerminalTask("task-1", "session-1", taskdomain.TaskStatusSuccess, "release", "release done", now.Add(-2*time.Hour)))
	store.Record(buildTerminalTask("task-2", "session-2", taskdomain.TaskStatusSuccess, "ops", "ops done", now.Add(-time.Hour)))

	overridden := store.Snapshot("不要查历史，只看当前任务", now)
	if overridden.DeepTriggered {
		t.Fatalf("expected deep retrieval blocked by negation")
	}
	if !overridden.DeepOverridden {
		t.Fatalf("expected deep retrieval overridden metric flag")
	}
	if overridden.Mode != RetrievalModeRecent {
		t.Fatalf("expected recent mode when overridden, got %q", overridden.Mode)
	}

	miss := store.Snapshot("请回顾 2025-01-01 的任务历史", now)
	if !miss.DeepTriggered {
		t.Fatalf("expected deep retrieval triggered by explicit date")
	}
	if !miss.DeepMiss {
		t.Fatalf("expected deep retrieval miss flag")
	}
	if miss.Mode != RetrievalModeFallback {
		t.Fatalf("expected fallback mode on deep miss, got %q", miss.Mode)
	}
	if len(miss.Summaries) == 0 {
		t.Fatalf("expected fallback summaries when miss")
	}
}

func buildTerminalTask(taskID string, sessionID string, status taskdomain.TaskStatus, taskType string, result string, finishedAt time.Time) taskdomain.Task {
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	return taskdomain.Task{
		ID:             taskID,
		SessionID:      sessionID,
		MessageID:      "msg-" + taskID,
		Status:         status,
		Progress:       100,
		RetryCount:     0,
		MaxRetries:     1,
		TimeoutMS:      90000,
		CreatedAt:      finishedAt.Add(-time.Minute),
		UpdatedAt:      finishedAt,
		FinishedAt:     finishedAt,
		RequestContent: "handle task " + taskID,
		RequestMetadata: map[string]string{
			"alter0.task.type": taskType,
		},
		Summary: "summary for " + taskID,
		TaskSummary: taskdomain.TaskSummary{
			TaskID:     taskID,
			TaskType:   taskType,
			Goal:       "handle task " + taskID,
			Result:     result,
			Status:     status,
			FinishedAt: finishedAt,
			Tags:       []string{"task", string(status), taskType},
		},
	}
}
