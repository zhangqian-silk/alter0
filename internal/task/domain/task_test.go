package domain

import (
	"strings"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

func TestTaskStatusAndLogLevelClassifiers(t *testing.T) {
	for _, status := range []TaskStatus{TaskStatusQueued, TaskStatusRunning, TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled} {
		if !status.IsValid() {
			t.Fatalf("%q IsValid() = false, want true", status)
		}
	}
	for _, status := range []TaskStatus{TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled} {
		if !status.IsTerminal() {
			t.Fatalf("%q IsTerminal() = false, want true", status)
		}
	}
	for _, status := range []TaskStatus{TaskStatusQueued, TaskStatusRunning} {
		if status.IsTerminal() {
			t.Fatalf("%q IsTerminal() = true, want false", status)
		}
	}
	for _, level := range []TaskLogLevel{TaskLogLevelInfo, TaskLogLevelWarn, TaskLogLevelError} {
		if !level.IsValid() {
			t.Fatalf("%q IsValid() = false, want true", level)
		}
	}
	if (TaskStatus("paused")).IsValid() {
		t.Fatalf("paused status IsValid() = true, want false")
	}
	if (TaskLogLevel("debug")).IsValid() {
		t.Fatalf("debug log level IsValid() = true, want false")
	}
}

func TestTaskSummaryZeroAndValidate(t *testing.T) {
	if !(TaskSummary{}).IsZero() {
		t.Fatalf("empty TaskSummary IsZero() = false, want true")
	}

	summary := TaskSummary{
		TaskID:     "task-1",
		TaskType:   "coding",
		Goal:       "ship tests",
		Result:     "done",
		Status:     TaskStatusSuccess,
		FinishedAt: time.Now().UTC(),
		Tags:       []string{"tests"},
	}
	if summary.IsZero() {
		t.Fatalf("populated TaskSummary IsZero() = true, want false")
	}
	if err := summary.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(TaskSummary) TaskSummary
		wantErr string
	}{
		{name: "task id", mutate: func(s TaskSummary) TaskSummary { s.TaskID = " "; return s }, wantErr: "task_summary.task_id is required"},
		{name: "task type", mutate: func(s TaskSummary) TaskSummary { s.TaskType = " "; return s }, wantErr: "task_summary.task_type is required"},
		{name: "goal", mutate: func(s TaskSummary) TaskSummary { s.Goal = " "; return s }, wantErr: "task_summary.goal is required"},
		{name: "result", mutate: func(s TaskSummary) TaskSummary { s.Result = " "; return s }, wantErr: "task_summary.result is required"},
		{name: "status", mutate: func(s TaskSummary) TaskSummary { s.Status = "paused"; return s }, wantErr: "task_summary.status is invalid"},
		{name: "finished", mutate: func(s TaskSummary) TaskSummary { s.FinishedAt = time.Time{}; return s }, wantErr: "task_summary.finished_at is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(summary).Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want containing %q", tc.wantErr)
			}
			assertTaskErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestTaskValidateAcceptsMessageIDFallbackAndNestedObjects(t *testing.T) {
	task := validTask()
	task.SourceMessageID = ""
	task.MessageID = "message-fallback"
	task.Logs = []TaskLog{
		{Seq: 0, CreatedAt: task.CreatedAt, Level: TaskLogLevelInfo, Message: "accepted"},
		{Seq: 1, Timestamp: task.CreatedAt, Level: TaskLogLevelWarn, Message: "running"},
	}
	task.Artifacts = []TaskArtifact{
		{ArtifactID: "artifact-1", Name: "result", ContentType: "text/plain", Size: 12, CreatedAt: task.CreatedAt},
		{URI: "memory://artifact", Name: "memory", ContentType: "text/markdown", Size: 0, CreatedAt: task.CreatedAt},
	}
	task.TaskSummary = TaskSummary{
		TaskID:     task.ID,
		TaskType:   "coding",
		Goal:       "ship tests",
		Result:     "done",
		Status:     TaskStatusSuccess,
		FinishedAt: task.CreatedAt,
	}
	task.MessageLink = TaskMessageLink{TaskID: task.ID, SessionID: task.SessionID}

	if err := task.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestTaskValidateRejectsInvalidBoundaries(t *testing.T) {
	base := validTask()
	cases := []struct {
		name    string
		mutate  func(Task) Task
		wantErr string
	}{
		{name: "blank id", mutate: func(t Task) Task { t.ID = " "; return t }, wantErr: "task id is required"},
		{name: "blank session", mutate: func(t Task) Task { t.SessionID = " "; return t }, wantErr: "session_id is required"},
		{name: "missing source message", mutate: func(t Task) Task { t.SourceMessageID = " "; t.MessageID = " "; return t }, wantErr: "source_message_id is required"},
		{name: "status", mutate: func(t Task) Task { t.Status = "paused"; return t }, wantErr: "task status is invalid"},
		{name: "phase", mutate: func(t Task) Task { t.Phase = " "; return t }, wantErr: "phase is required"},
		{name: "progress negative", mutate: func(t Task) Task { t.Progress = -1; return t }, wantErr: "progress must be within 0..100"},
		{name: "progress over 100", mutate: func(t Task) Task { t.Progress = 101; return t }, wantErr: "progress must be within 0..100"},
		{name: "queue position negative", mutate: func(t Task) Task { t.QueuePosition = -1; return t }, wantErr: "queue_position must be non-negative"},
		{name: "queue wait negative", mutate: func(t Task) Task { t.QueueWaitMS = -1; return t }, wantErr: "queue_wait_ms must be non-negative"},
		{name: "queue position on running", mutate: func(t Task) Task { t.Status = TaskStatusRunning; t.QueuePosition = 1; return t }, wantErr: "queue_position is only valid for queued status"},
		{name: "retry count", mutate: func(t Task) Task { t.RetryCount = -1; return t }, wantErr: "retry_count must be non-negative"},
		{name: "max retries", mutate: func(t Task) Task { t.MaxRetries = -1; return t }, wantErr: "max_retries must be non-negative"},
		{name: "timeout", mutate: func(t Task) Task { t.TimeoutMS = -1; return t }, wantErr: "timeout_ms must be non-negative"},
		{name: "created", mutate: func(t Task) Task { t.CreatedAt = time.Time{}; return t }, wantErr: "created_at is required"},
		{name: "updated", mutate: func(t Task) Task { t.UpdatedAt = time.Time{}; return t }, wantErr: "updated_at is required"},
		{name: "request", mutate: func(t Task) Task { t.RequestContent = " "; return t }, wantErr: "request_content is required"},
		{name: "log seq", mutate: func(t Task) Task {
			t.Logs = []TaskLog{{Seq: -1, CreatedAt: t.CreatedAt, Level: TaskLogLevelInfo, Message: "x"}}
			return t
		}, wantErr: "task log seq must be non-negative"},
		{name: "log timestamp", mutate: func(t Task) Task { t.Logs = []TaskLog{{Seq: 1, Level: TaskLogLevelInfo, Message: "x"}}; return t }, wantErr: "task log created_at is required"},
		{name: "log level", mutate: func(t Task) Task {
			t.Logs = []TaskLog{{Seq: 1, CreatedAt: t.CreatedAt, Level: "debug", Message: "x"}}
			return t
		}, wantErr: "task log level is invalid"},
		{name: "log message", mutate: func(t Task) Task {
			t.Logs = []TaskLog{{Seq: 1, CreatedAt: t.CreatedAt, Level: TaskLogLevelInfo, Message: " "}}
			return t
		}, wantErr: "task log message is required"},
		{name: "artifact id", mutate: func(t Task) Task {
			t.Artifacts = []TaskArtifact{{Name: "x", ContentType: "text/plain", CreatedAt: t.CreatedAt}}
			return t
		}, wantErr: "artifact_id is required"},
		{name: "artifact name", mutate: func(t Task) Task {
			t.Artifacts = []TaskArtifact{{ArtifactID: "a1", ContentType: "text/plain", CreatedAt: t.CreatedAt}}
			return t
		}, wantErr: "artifact name is required"},
		{name: "artifact content type", mutate: func(t Task) Task {
			t.Artifacts = []TaskArtifact{{ArtifactID: "a1", Name: "x", CreatedAt: t.CreatedAt}}
			return t
		}, wantErr: "artifact content_type is required"},
		{name: "artifact size", mutate: func(t Task) Task {
			t.Artifacts = []TaskArtifact{{ArtifactID: "a1", Name: "x", ContentType: "text/plain", Size: -1, CreatedAt: t.CreatedAt}}
			return t
		}, wantErr: "artifact size must be non-negative"},
		{name: "artifact created", mutate: func(t Task) Task {
			t.Artifacts = []TaskArtifact{{ArtifactID: "a1", Name: "x", ContentType: "text/plain"}}
			return t
		}, wantErr: "artifact created_at is required"},
		{name: "summary invalid", mutate: func(t Task) Task { t.TaskSummary = TaskSummary{TaskID: t.ID}; return t }, wantErr: "task summary invalid"},
		{name: "summary mismatch", mutate: func(t Task) Task {
			t.TaskSummary = TaskSummary{TaskID: "other", TaskType: "coding", Goal: "g", Result: "r", Status: TaskStatusSuccess, FinishedAt: t.CreatedAt}
			return t
		}, wantErr: "task_summary.task_id must match task id"},
		{name: "message task mismatch", mutate: func(t Task) Task { t.MessageLink.TaskID = "other"; return t }, wantErr: "message_link.task_id must match task id"},
		{name: "message session mismatch", mutate: func(t Task) Task { t.MessageLink.SessionID = "other"; return t }, wantErr: "message_link.session_id must match session id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(base).Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want containing %q", tc.wantErr)
			}
			assertTaskErrorContains(t, err, tc.wantErr)
		})
	}
}

func validTask() Task {
	now := time.Date(2026, 4, 8, 8, 30, 0, 0, time.UTC)
	return Task{
		ID:              "task-1",
		SessionID:       "session-1",
		SourceMessageID: "message-1",
		TaskType:        "coding",
		Status:          TaskStatusQueued,
		Phase:           "queued",
		Progress:        0,
		CreatedAt:       now,
		UpdatedAt:       now,
		RequestContent:  "write tests",
		Result:          TaskResult{Route: shareddomain.RouteNL, Output: "ok"},
		MessageLink:     TaskMessageLink{TaskID: "task-1", SessionID: "session-1"},
	}
}

func assertTaskErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
}
