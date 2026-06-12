package application

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	taskdomain "alter0/internal/task/domain"
)

func TestRuntimeMarkdownStoreRecordDoesNotWriteMemoryFilesDirectly(t *testing.T) {
	root := t.TempDir()
	dailyDir := filepath.Join(root, "memory")
	longTermDir := filepath.Join(root, "memory", "long-term")
	store := NewRuntimeMarkdownStore(RuntimeMarkdownOptions{
		DailyDir:    dailyDir,
		LongTermDir: longTermDir,
	})

	finished := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	task := buildTerminalSummaryTask("task-r032-1", taskdomain.TaskStatusSuccess, finished, "initial")
	store.Record(task)

	dailyPath := filepath.Join(dailyDir, "2026-03-04.md")
	longTermPath := filepath.Join(longTermDir, "2026-03-04.md")
	if _, err := os.Stat(dailyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected task summary not to write daily memory directly, got %v", err)
	}
	if _, err := os.Stat(longTermPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected task summary not to write long-term memory directly, got %v", err)
	}
	refs := store.FindSummaryRefs("task-r032-1")
	if len(refs) != 0 {
		t.Fatalf("expected no markdown summary refs, got %+v", refs)
	}
}

func TestRuntimeMarkdownStoreRebuildReturnsNoMemoryRefs(t *testing.T) {
	root := t.TempDir()
	dailyDir := filepath.Join(root, "memory")
	longTermDir := filepath.Join(root, "memory", "long-term")
	store := NewRuntimeMarkdownStore(RuntimeMarkdownOptions{
		DailyDir:    dailyDir,
		LongTermDir: longTermDir,
	})

	finished := time.Date(2026, 3, 4, 11, 0, 0, 0, time.UTC)
	first := buildTerminalSummaryTask("task-r032-2", taskdomain.TaskStatusFailed, finished, "first")
	refs, err := store.Rebuild(first)
	if err != nil {
		t.Fatalf("first rebuild: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected no direct memory refs, got %+v", refs)
	}
	updated := first
	updated.TaskSummary.Result = "updated"
	updated.Summary = "updated"
	refs, err = store.Rebuild(updated)
	if err != nil {
		t.Fatalf("second rebuild: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected no direct memory refs after rebuild, got %+v", refs)
	}
}

func TestRecorderGroupFanout(t *testing.T) {
	r1 := &stubTaskRecorder{}
	r2 := &stubTaskRecorder{}
	group := NewRecorderGroup(r1, r2)
	group.Record(buildTerminalSummaryTask("task-r032-3", taskdomain.TaskStatusSuccess, time.Now().UTC(), "ok"))
	if r1.called != 1 || r2.called != 1 {
		t.Fatalf("expected record fanout once, got r1=%d r2=%d", r1.called, r2.called)
	}
}

type stubTaskRecorder struct {
	called int
}

func (s *stubTaskRecorder) Record(task taskdomain.Task) {
	if strings.TrimSpace(task.ID) != "" {
		s.called++
	}
}

func buildTerminalSummaryTask(taskID string, status taskdomain.TaskStatus, finishedAt time.Time, result string) taskdomain.Task {
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	return taskdomain.Task{
		ID:              taskID,
		SessionID:       "session-1",
		SourceMessageID: "msg-1",
		MessageID:       "msg-1",
		TaskType:        "task",
		Status:          status,
		Progress:        100,
		MaxRetries:      1,
		TimeoutMS:       60000,
		CreatedAt:       finishedAt.Add(-time.Minute),
		UpdatedAt:       finishedAt,
		FinishedAt:      finishedAt,
		RequestContent:  "build release",
		Summary:         result,
		TaskSummary: taskdomain.TaskSummary{
			TaskID:     taskID,
			TaskType:   "task",
			Goal:       "build release",
			Result:     result,
			Status:     status,
			FinishedAt: finishedAt,
			Tags:       []string{"task", string(status)},
		},
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path %s exists: %v", path, err)
	}
}
