package localfile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
	taskdomain "alter0/internal/task/domain"
)

func TestTaskStorePersistsTaskLayoutRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskStore(dir, FormatJSON)
	base := time.Date(2026, 3, 4, 1, 2, 3, 0, time.UTC)
	input := []taskdomain.Task{
		{
			ID:              "task-1",
			SessionID:       "session-1",
			SourceMessageID: "message-1",
			MessageID:       "message-1",
			TaskType:        "artifact",
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
				{Seq: 1, Timestamp: base, CreatedAt: base, Level: taskdomain.TaskLogLevelInfo, Message: "accepted"},
			},
			Artifacts: []taskdomain.TaskArtifact{
				{
					ArtifactID:   "task-1-result",
					ArtifactType: "report",
					Name:         "result.txt",
					ContentType:  "text/plain",
					Size:         int64(len([]byte("report ready"))),
					Content:      "report ready",
					URI:          "inline://result.txt",
					CreatedAt:    base.Add(4 * time.Second),
				},
			},
		},
	}
	if err := store.Save(context.Background(), input); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	assertFileExists(t, filepath.Join(dir, "tasks", "index.json"))
	assertFileExists(t, filepath.Join(dir, "tasks", "task-1", "meta.json"))
	assertFileExists(t, filepath.Join(dir, "tasks", "task-1", "logs.jsonl"))
	assertFileExists(t, filepath.Join(dir, "tasks", "task-1", "artifacts.json"))
	assertFileExists(t, filepath.Join(dir, "tasks", "task-1", "files", "task-1-result"))

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
	if item.SourceMessageID != "message-1" {
		t.Fatalf("expected source_message_id message-1, got %q", item.SourceMessageID)
	}
	if len(item.Logs) != 1 || item.Logs[0].Message != "accepted" {
		t.Fatalf("unexpected logs: %+v", item.Logs)
	}
	if len(item.Artifacts) != 1 || item.Artifacts[0].ArtifactID != "task-1-result" {
		t.Fatalf("unexpected artifacts: %+v", item.Artifacts)
	}
	if item.Artifacts[0].DownloadURL == "" {
		t.Fatalf("expected artifact download url, got %+v", item.Artifacts[0])
	}
	if item.Artifacts[0].URI != "" {
		t.Fatalf("expected persisted artifact uri hidden, got %+v", item.Artifacts[0])
	}
}

func TestTaskStoreReadArtifactFromSnapshot(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskStore(dir, FormatJSON)
	now := time.Date(2026, 3, 4, 2, 2, 2, 0, time.UTC)
	task := taskdomain.Task{
		ID:              "task-artifact-read",
		SessionID:       "session-artifact-read",
		SourceMessageID: "message-artifact-read",
		MessageID:       "message-artifact-read",
		Status:          taskdomain.TaskStatusSuccess,
		Progress:        100,
		MaxRetries:      1,
		TimeoutMS:       60000,
		CreatedAt:       now,
		UpdatedAt:       now,
		FinishedAt:      now,
		RequestContent:  "generate artifact",
		Artifacts: []taskdomain.TaskArtifact{
			{
				ArtifactID:  "artifact-a",
				Name:        "artifact.txt",
				ContentType: "text/plain",
				Content:     "artifact content",
				CreatedAt:   now,
			},
		},
	}
	if err := store.Save(context.Background(), []taskdomain.Task{task}); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	artifact, raw, err := store.ReadArtifact(context.Background(), "task-artifact-read", "artifact-a")
	if err != nil {
		t.Fatalf("read artifact failed: %v", err)
	}
	if artifact.DownloadURL == "" {
		t.Fatalf("expected download_url in artifact metadata, got %+v", artifact)
	}
	if got := string(raw); got != "artifact content" {
		t.Fatalf("expected artifact content, got %q", got)
	}
}

func TestTaskStoreAppendsLogsWithoutDuplication(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskStore(dir, FormatJSON)
	base := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	task := taskdomain.Task{
		ID:              "task-2",
		SessionID:       "session-2",
		SourceMessageID: "message-2",
		MessageID:       "message-2",
		Status:          taskdomain.TaskStatusRunning,
		Progress:        50,
		MaxRetries:      1,
		TimeoutMS:       5000,
		CreatedAt:       base,
		UpdatedAt:       base,
		RequestContent:  "export artifact",
		Logs: []taskdomain.TaskLog{
			{Seq: 1, Timestamp: base, CreatedAt: base, Level: taskdomain.TaskLogLevelInfo, Message: "accepted"},
		},
	}
	if err := store.Save(context.Background(), []taskdomain.Task{task}); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	task.Logs = append(task.Logs, taskdomain.TaskLog{Seq: 2, Timestamp: base.Add(time.Second), CreatedAt: base.Add(time.Second), Level: taskdomain.TaskLogLevelInfo, Message: "running"})
	task.UpdatedAt = base.Add(time.Second)
	if err := store.Save(context.Background(), []taskdomain.Task{task}); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "tasks", "task-2", "logs.jsonl"))
	if err != nil {
		t.Fatalf("read logs jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 || len(loaded[0].Logs) != 2 {
		t.Fatalf("expected 2 logs after load, got %+v", loaded)
	}
}

func TestTaskStoreMigratesLegacyTaskFile(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "tasks.md")
	base := time.Date(2026, 3, 4, 8, 0, 0, 0, time.UTC)
	payload, err := marshalPayload(FormatMarkdown, "alter0 async task state", taskState{Tasks: []taskdomain.Task{
		{
			ID:              "task-legacy",
			SessionID:       "session-legacy",
			SourceMessageID: "message-legacy",
			MessageID:       "message-legacy",
			Status:          taskdomain.TaskStatusSuccess,
			Progress:        100,
			MaxRetries:      1,
			TimeoutMS:       60000,
			CreatedAt:       base,
			UpdatedAt:       base,
			FinishedAt:      base,
			RequestContent:  "legacy payload",
		},
	}})
	if err != nil {
		t.Fatalf("marshal legacy payload: %v", err)
	}
	if err := os.WriteFile(legacyPath, payload, 0o644); err != nil {
		t.Fatalf("write legacy task file: %v", err)
	}

	store := NewTaskStore(dir, FormatMarkdown)
	items, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load from legacy file failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "task-legacy" {
		t.Fatalf("unexpected migrated tasks: %+v", items)
	}
	assertFileExists(t, filepath.Join(dir, "tasks", "index.json"))
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy file removed after migration")
	}
}

func TestTaskStoreLogRetention(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskStoreWithOptions(dir, FormatJSON, TaskStoreOptions{LogRetention: 24 * time.Hour})
	now := time.Now().UTC()

	task := taskdomain.Task{
		ID:              "task-retention",
		SessionID:       "session-retention",
		SourceMessageID: "message-retention",
		MessageID:       "message-retention",
		Status:          taskdomain.TaskStatusSuccess,
		Progress:        100,
		MaxRetries:      1,
		TimeoutMS:       60000,
		CreatedAt:       now.Add(-48 * time.Hour),
		UpdatedAt:       now,
		FinishedAt:      now,
		RequestContent:  "retention",
		Logs: []taskdomain.TaskLog{
			{Seq: 1, Timestamp: now.Add(-48 * time.Hour), CreatedAt: now.Add(-48 * time.Hour), Level: taskdomain.TaskLogLevelInfo, Message: "old"},
			{Seq: 2, Timestamp: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour), Level: taskdomain.TaskLogLevelInfo, Message: "fresh"},
		},
	}
	if err := store.Save(context.Background(), []taskdomain.Task{task}); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected one task, got %d", len(loaded))
	}
	if len(loaded[0].Logs) != 1 || loaded[0].Logs[0].Message != "fresh" {
		t.Fatalf("expected retention filtered logs, got %+v", loaded[0].Logs)
	}
}

func TestTaskStoreSaveRemovesDeletedTaskDirectories(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskStore(dir, FormatJSON)
	now := time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)

	first := taskdomain.Task{
		ID:              "task-keep",
		SessionID:       "session-1",
		SourceMessageID: "message-1",
		MessageID:       "message-1",
		Status:          taskdomain.TaskStatusSuccess,
		Progress:        100,
		MaxRetries:      1,
		TimeoutMS:       60000,
		CreatedAt:       now,
		UpdatedAt:       now,
		FinishedAt:      now,
		RequestContent:  "keep",
	}
	second := taskdomain.Task{
		ID:              "task-delete",
		SessionID:       "session-1",
		SourceMessageID: "message-2",
		MessageID:       "message-2",
		Status:          taskdomain.TaskStatusSuccess,
		Progress:        100,
		MaxRetries:      1,
		TimeoutMS:       60000,
		CreatedAt:       now.Add(time.Second),
		UpdatedAt:       now.Add(time.Second),
		FinishedAt:      now.Add(time.Second),
		RequestContent:  "delete",
	}

	if err := store.Save(context.Background(), []taskdomain.Task{first, second}); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}
	if err := store.Save(context.Background(), []taskdomain.Task{first}); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "tasks", "task-delete")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted task directory removed, got %v", err)
	}
	assertFileExists(t, filepath.Join(dir, "tasks", "task-keep", "meta.json"))
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s exists: %v", path, err)
	}
}
