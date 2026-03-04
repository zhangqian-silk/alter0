package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	taskdomain "alter0/internal/task/domain"
)

func TestRuntimeMarkdownStoreRecordWritesDailyAndLongTerm(t *testing.T) {
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
	assertPathExists(t, dailyPath)
	assertPathExists(t, longTermPath)

	dailyRaw, err := os.ReadFile(dailyPath)
	if err != nil {
		t.Fatalf("read daily summary: %v", err)
	}
	if !strings.Contains(string(dailyRaw), "task_id: task-r032-1") {
		t.Fatalf("expected task_id reference in daily summary, got %s", string(dailyRaw))
	}
	refs := store.FindSummaryRefs("task-r032-1")
	if len(refs) != 2 {
		t.Fatalf("expected two summary refs, got %+v", refs)
	}
}

func TestRuntimeMarkdownStoreRebuildUpsertsEntry(t *testing.T) {
	root := t.TempDir()
	dailyDir := filepath.Join(root, "memory")
	longTermDir := filepath.Join(root, "memory", "long-term")
	store := NewRuntimeMarkdownStore(RuntimeMarkdownOptions{
		DailyDir:    dailyDir,
		LongTermDir: longTermDir,
	})

	finished := time.Date(2026, 3, 4, 11, 0, 0, 0, time.UTC)
	first := buildTerminalSummaryTask("task-r032-2", taskdomain.TaskStatusFailed, finished, "first")
	if _, err := store.Rebuild(first); err != nil {
		t.Fatalf("first rebuild: %v", err)
	}
	updated := first
	updated.TaskSummary.Result = "updated"
	updated.Summary = "updated"
	if _, err := store.Rebuild(updated); err != nil {
		t.Fatalf("second rebuild: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dailyDir, "2026-03-04.md"))
	if err != nil {
		t.Fatalf("read summary file: %v", err)
	}
	body := string(raw)
	if strings.Count(body, "### task_id: task-r032-2") != 1 {
		t.Fatalf("expected unique task entry, got %s", body)
	}
	if !strings.Contains(body, "- result: updated") {
		t.Fatalf("expected updated summary content, got %s", body)
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
