package localfile

import (
	"context"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
	taskdomain "alter0/internal/task/domain"
)

func TestTaskStoreJSONRoundTrip(t *testing.T) {
	store := NewTaskStore(t.TempDir(), FormatJSON)
	base := time.Date(2026, 3, 4, 1, 2, 3, 0, time.UTC)
	input := []taskdomain.Task{
		{
			ID:              "task-1",
			SessionID:       "session-1",
			MessageID:       "message-1",
			Status:          taskdomain.TaskStatusSuccess,
			Progress:        100,
			MaxRetries:      1,
			TimeoutMS:       60000,
			CreatedAt:       base,
			UpdatedAt:       base.Add(4 * time.Second),
			StartedAt:       base.Add(1 * time.Second),
			FinishedAt:      base.Add(4 * time.Second),
			RequestContent:  "generate report",
			RequestMetadata: map[string]string{"k": "v"},
			Summary:         "completed",
			Result: taskdomain.TaskResult{
				Route:    shareddomain.RouteNL,
				Output:   "report ready",
				Metadata: map[string]string{"task_id": "task-1"},
			},
			Logs: []taskdomain.TaskLog{
				{Timestamp: base, Level: taskdomain.TaskLogLevelInfo, Message: "accepted"},
			},
			Artifacts: []taskdomain.TaskArtifact{
				{
					ArtifactID:  "task-1-result",
					Name:        "result.txt",
					ContentType: "text/plain",
					Content:     "report ready",
					CreatedAt:   base.Add(4 * time.Second),
				},
			},
		},
	}
	if err := store.Save(context.Background(), input); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 task, got %d", len(loaded))
	}
	item := loaded[0]
	if item.ID != "task-1" || item.SessionID != "session-1" {
		t.Fatalf("unexpected task id/session: %+v", item)
	}
	if item.Status != taskdomain.TaskStatusSuccess {
		t.Fatalf("expected success status, got %q", item.Status)
	}
	if len(item.Logs) != 1 || item.Logs[0].Message != "accepted" {
		t.Fatalf("unexpected logs: %+v", item.Logs)
	}
	if len(item.Artifacts) != 1 || item.Artifacts[0].ArtifactID != "task-1-result" {
		t.Fatalf("unexpected artifacts: %+v", item.Artifacts)
	}
}

func TestTaskStoreMarkdownRoundTrip(t *testing.T) {
	store := NewTaskStore(t.TempDir(), FormatMarkdown)
	base := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	input := []taskdomain.Task{
		{
			ID:             "task-2",
			SessionID:      "session-2",
			MessageID:      "message-2",
			Status:         taskdomain.TaskStatusFailed,
			Progress:       100,
			RetryCount:     1,
			MaxRetries:     1,
			TimeoutMS:      5000,
			CreatedAt:      base,
			UpdatedAt:      base.Add(5 * time.Second),
			StartedAt:      base.Add(1 * time.Second),
			FinishedAt:     base.Add(5 * time.Second),
			RequestContent: "export artifact",
			Summary:        "failed",
			ErrorCode:      "task_timeout",
			ErrorMessage:   "deadline exceeded",
			Result: taskdomain.TaskResult{
				Route:     shareddomain.RouteNL,
				ErrorCode: "task_timeout",
			},
			Logs: []taskdomain.TaskLog{
				{Timestamp: base.Add(5 * time.Second), Level: taskdomain.TaskLogLevelError, Message: "timed out"},
			},
		},
	}
	if err := store.Save(context.Background(), input); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 task, got %d", len(loaded))
	}
	item := loaded[0]
	if item.ErrorCode != "task_timeout" {
		t.Fatalf("expected task_timeout, got %q", item.ErrorCode)
	}
	if len(item.Logs) != 1 || item.Logs[0].Level != taskdomain.TaskLogLevelError {
		t.Fatalf("unexpected logs: %+v", item.Logs)
	}
}
